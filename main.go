package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/opentracing/opentracing-go/ext"

	"github.com/gin-gonic/gin"
	"github.com/ishrivatsa/catalogservice/catalog"
	stdopentracing "github.com/opentracing/opentracing-go"
	zipkintracer "github.com/openzipkin/zipkin-go-opentracing"
	"github.com/sirupsen/logrus"
)

var (
	logger      *logrus.Logger
	zip         = flag.String("zipkin", os.Getenv("ZIPKIN"), "Zipkin address")
	serviceName = "catalog"
)

const (
	dbName         = "catalog"
	collectionName = "products"
)

// // Middleware to handle spans
// func NewSpan(tracer stdopentracing.Tracer, operationName string, opts ...stdopentracing.StartSpanOption) gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		span := tracer.StartSpan(operationName, opts...)
// 		c.Set("span", span)
// 		defer span.Finish()
// 		c.Next()
// 	}
// }

func OpenTracing() gin.HandlerFunc {
	return func(c *gin.Context) {
		wireCtx, _ := stdopentracing.GlobalTracer().Extract(
			stdopentracing.HTTPHeaders,
			stdopentracing.HTTPHeadersCarrier(c.Request.Header))

		serverSpan := stdopentracing.StartSpan(c.Request.URL.Path,
			ext.RPCServerOption(wireCtx))

		fmt.Println(c.Request.Context())
		defer serverSpan.Finish()
		c.Request = c.Request.WithContext(stdopentracing.ContextWithSpan(c.Request.Context(), serverSpan))
		fmt.Println("/n This is c.request /n")
		fmt.Println(c.Request)
		c.Next()
	}
}

func handleRequest(tracer stdopentracing.Tracer) {

	router := gin.Default()

	router.Use(OpenTracing())

	router.Static("/static/images", "./images")

	v1 := router.Group("/")
	{
		v1.GET("/products", catalog.GetProducts)
		v1.GET("/products/:id", catalog.GetProduct)
		//v1.POST("/products", catalog.CreateProduct)
	}

	router.Run(":8080")
}

func initLogger(f *os.File) {

	logger = logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "",
		PrettyPrint:     true,
	})

	//set output of logs to f
	logger.SetOutput(f)

}

func main() {

	//create your file with desired read/write permissions
	f, err := os.OpenFile("log.info", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println("Could not open file ", err)
	} else {
		initLogger(f)
	}

	dbsession := catalog.ConnectDB(dbName, collectionName, logger)

	logger.Infof("Successfully connected to database %s", dbName)

	// @todo: Replace with Jaeger
	zipkinCollector, err := zipkintracer.NewHTTPCollector("http://0.0.0.0:9411/api/v1/spans")
	if err != nil {
		logger.Fatalf("unable to create Zipkin HTTP collector: %+v", err)
	}
	defer zipkinCollector.Close()

	zipkinRecorder := zipkintracer.NewRecorder(zipkinCollector, false, "0.0.0.0:8080", "catalog")
	tracer, err := zipkintracer.NewTracer(zipkinRecorder, zipkintracer.ClientServerSameSpan(true), zipkintracer.TraceID128Bit(true))
	if err != nil {
		logger.Fatalf("unable to create Zipkin tracer: %+v", err)
	}

	stdopentracing.SetGlobalTracer(tracer)

	handleRequest(tracer)

	catalog.CloseDB(dbsession, logger)

	// defer to close
	defer f.Close()
}
