package catalog

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
	stdopentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
)

// Product struct
type Product struct {
	ID          bson.ObjectId `json:"id" bson:"_id,omitempty"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Picture     string        `json:"imageUrl"`
	Price       float32       `json:"price"`
	Tags        []string      `json:"tag"`
}

// GetProducts returns a list of all products
func GetProducts(c *gin.Context) {
	var products []Product

	ctx, _ := stdopentracing.GlobalTracer().Extract(
		stdopentracing.HTTPHeaders,
		stdopentracing.HTTPHeadersCarrier(c.Request.Header))
	// // iSpan, _ := c.Get("span")
	// // span, found := iSpan.(stdopentracing.Span)
	// // if !(span != nil && found) == false {
	// // 	c.AbortWithStatus(http.StatusInternalServerError)
	// // }

	//	spanCtx := tracer.Extract

	fmt.Println(ctx)

	// span := opentracing.StartSpan(c.Request.URL.Path,
	// 	ext.RPCServerOption(ctx))

	textspan := stdopentracing.ChildOf(ctx)
	fmt.Println(textspan)

	span := stdopentracing.StartSpan("Accessing DB", stdopentracing.ChildOf(ctx))
	span.SetTag("database access", "access")

	defer span.Finish()

	span.LogFields(
		log.String("event", "string-format"),
	)

	error := collection.Find(nil).All(&products)

	if error != nil {
		message := "Products " + error.Error()
		ext.Error.Set(span, true) // Tag the span as errored
		span.LogEventWithPayload("GET service error", message)
		c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": message})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": products})

}

// GetProduct returns a single product based on id
func GetProduct(c *gin.Context) {
	var product Product

	span, _ := stdopentracing.StartSpanFromContext(c, "get_product")
	defer span.Finish()

	productID := c.Param("id")

	span.LogFields(
		log.String("event", "string-format"),
		log.String("ProductID", productID),
	)

	if bson.IsObjectIdHex(productID) {
		error := collection.FindId(bson.ObjectIdHex(productID)).One(&product)

		if error != nil {
			message := "Product " + error.Error()
			ext.Error.Set(span, true) // Tag the span as errored
			span.LogEventWithPayload("GET product error", message)
			c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": message})
			return
		}

	} else {
		message := "Incorrect Format for ProductID"
		ext.Error.Set(span, true) // Tag the span as errored
		span.LogEventWithPayload("Incorrect Format for ProductID", message)
		c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": message})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": product})

}

// CreateProduct adds a new product item to the database
func CreateProduct(c *gin.Context) {
	var product Product

	error := c.ShouldBindJSON(&product)

	if error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": "Incorrect Field Name(s)/ Value(s)"})
		return
	}

	product.ID = bson.NewObjectId()

	error = collection.Insert(&product)

	if error != nil {
		message := "Product " + error.Error()
		c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": message})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"status": http.StatusCreated, "message": "Product created successfully!", "resourceId": product})

}
