package main

import (
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Product struct {
	ProductID    int    `json:"product_id"`
	SKU          string `json:"sku"`
	Manufacturer string `json:"manufacturer"`
	CategoryID   int    `json:"category_id"`
	Weight       int    `json:"weight"`
	SomeOtherID  int    `json:"some_other_id"`
}

type ProductInput struct {
	ProductID    int    `json:"product_id" binding:"required,min=1"`
	SKU          string `json:"sku" binding:"required,min=1,max=100"`
	Manufacturer string `json:"manufacturer" binding:"required,min=1,max=200"`
	CategoryID   int    `json:"category_id" binding:"required,min=1"`
	Weight       *int   `json:"weight" binding:"required,gte=0"`
	SomeOtherID  int    `json:"some_other_id" binding:"required,min=1"`
}

type APIError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

var (
	products = map[int]Product{
		12345: {
			ProductID:    12345,
			SKU:          "ABC-123-XYZ",
			Manufacturer: "Acme Corporation",
			CategoryID:   456,
			Weight:       1250,
			SomeOtherID:  789,
		},
	}

	mu sync.RWMutex

	requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "product_api_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"method", "endpoint", "status"},
	)

	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "product_api_request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	requestErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "product_api_errors_total",
			Help: "Total number of HTTP requests returning error status codes.",
		},
		[]string{"method", "endpoint", "status"},
	)
)

func init() {
	prometheus.MustRegister(requestsTotal, requestDuration, requestErrors)
}

func metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start).Seconds()
		status := c.Writer.Status()
		method := c.Request.Method
		endpoint := c.FullPath()

		if endpoint == "" {
			endpoint = "unmatched"
		}

		statusText := strconv.Itoa(status)

		requestsTotal.WithLabelValues(
			method, endpoint, statusText,
		).Inc()

		requestDuration.WithLabelValues(
			method, endpoint,
		).Observe(duration)

		if status >= 400 {
			requestErrors.WithLabelValues(
				method, endpoint, statusText,
			).Inc()
		}

		log.Printf(
			`{"timestamp":"%s","method":"%s","endpoint":"%s","status":%d,"duration_seconds":%.6f}`,
			time.Now().UTC().Format(time.RFC3339Nano),
			method,
			endpoint,
			status,
			duration,
		)
	}
}

func errorResponse(
	c *gin.Context,
	status int,
	code, message, details string,
) {
	c.JSON(status, APIError{
		Error:   code,
		Message: message,
		Details: details,
	})
}

func getProduct(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("productId"))
	if err != nil || id < 1 {
		errorResponse(
			c, http.StatusNotFound, "PRODUCT_NOT_FOUND",
			"Product not found",
			"Product ID must be a positive integer",
		)
		return
	}

	mu.RLock()
	product, exists := products[id]
	mu.RUnlock()

	if !exists {
		errorResponse(
			c, http.StatusNotFound, "PRODUCT_NOT_FOUND",
			"Product not found",
			"No product exists with the supplied ID",
		)
		return
	}

	c.JSON(http.StatusOK, product)
}

func addProductDetails(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("productId"))
	if err != nil || id < 1 {
		errorResponse(
			c, http.StatusNotFound, "PRODUCT_NOT_FOUND",
			"Product not found",
			"Product ID must be a positive integer",
		)
		return
	}

	var input ProductInput
	if err := c.ShouldBindJSON(&input); err != nil {
		errorResponse(
			c, http.StatusBadRequest, "INVALID_INPUT",
			"The provided input data is invalid",
			err.Error(),
		)
		return
	}

	if input.ProductID != id {
		errorResponse(
			c, http.StatusBadRequest, "INVALID_INPUT",
			"The provided input data is invalid",
			"product_id must match the productId in the URL",
		)
		return
	}

	mu.Lock()
	defer mu.Unlock()

	if _, exists := products[id]; !exists {
		errorResponse(
			c, http.StatusNotFound, "PRODUCT_NOT_FOUND",
			"Product not found",
			"No product exists with the supplied ID",
		)
		return
	}

	products[id] = Product{
		ProductID:    input.ProductID,
		SKU:          input.SKU,
		Manufacturer: input.Manufacturer,
		CategoryID:   input.CategoryID,
		Weight:       *input.Weight,
		SomeOtherID:  input.SomeOtherID,
	}

	c.Status(http.StatusNoContent)
}

func main() {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("Internal server error: %v", recovered)
				c.AbortWithStatusJSON(http.StatusInternalServerError, APIError{
					Error:   "INTERNAL_SERVER_ERROR",
					Message: "An unexpected internal server error occurred",
				})
			}
		}()
		c.Next()
	})
	router.Use(metricsMiddleware())

	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	router.GET("/products/:productId", getProduct)
	router.POST("/products/:productId/details", addProductDetails)

	if err := router.Run(":8080"); err != nil {
		panic(err)
	}
}
