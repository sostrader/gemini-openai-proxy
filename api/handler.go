package api

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/generative-ai-go/genai"
	openai "github.com/sashabaranov/go-openai"
	"google.golang.org/api/option"

	"github.com/zhu327/gemini-openai-proxy/pkg/adapter"
)

func IndexHandler(c *gin.Context) {
	c.JSON(http.StatusMisdirectedRequest, gin.H{
		"message": "Welcome to the OpenAI API! Documentation is available at https://platform.openai.com/docs/api-reference",
	})
}

func ModelListHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data": []any{
			openai.Model{
				CreatedAt: 1686935002,
				ID:        adapter.Gemini1Pro,
				Object:    "model",
				OwnedBy:   "google",
			},
			openai.Model{
				CreatedAt: 1686935002,
				ID:        adapter.Gemini1Dot5Flash,
				Object:    "model",
				OwnedBy:   "google",
			},
			openai.Model{
				CreatedAt: 1686935002,
				ID:        adapter.Gemini1Dot5Pro,
				Object:    "model",
				OwnedBy:   "google",
			},
			openai.Model{
				CreatedAt: 1686935002,
				ID:        adapter.TextEmbedding004,
				Object:    "model",
				OwnedBy:   "openai",
			},
		},
	})
}

func ModelRetrieveHandler(c *gin.Context) {
	model := c.Param("model")
	c.JSON(http.StatusOK, openai.Model{
		CreatedAt: 1686935002,
		ID:        model,
		Object:    "model",
		OwnedBy:   "google",
	})
}

func ChatProxyHandler(c *gin.Context) {
	// Retrieve the Authorization header value
	authorizationHeader := c.GetHeader("Authorization")
	// Declare a variable to store the OPENAI_API_KEY
	var openaiAPIKey string
	// Use fmt.Sscanf to extract the Bearer token
	_, err := fmt.Sscanf(authorizationHeader, "Bearer %s", &openaiAPIKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, openai.APIError{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	req := &adapter.ChatCompletionRequest{}
	// Bind the JSON data from the request to the struct
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, openai.APIError{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	messages, err := req.ToGenaiMessages()
	if err != nil {
		c.JSON(http.StatusBadRequest, openai.APIError{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	ctx := c.Request.Context()
	client, err := genai.NewClient(ctx, option.WithAPIKey(openaiAPIKey))
	if err != nil {
		log.Printf("new genai client error %v\n", err)
		c.JSON(http.StatusBadRequest, openai.APIError{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}
	defer client.Close()

	model := req.Model
	gemini := adapter.NewGeminiAdapter(client, model)

	if !req.Stream {
		resp, err := gemini.GenerateContent(ctx, req, messages)
		if err != nil {
			log.Printf("genai generate content error %v\n", err)
			c.JSON(http.StatusBadRequest, openai.APIError{
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, resp)
		return
	}

	dataChan, err := gemini.GenerateStreamContent(ctx, req, messages)
	if err != nil {
		log.Printf("genai generate content error %v\n", err)
		c.JSON(http.StatusBadRequest, openai.APIError{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	setEventStreamHeaders(c)
	c.Stream(func(w io.Writer) bool {
		if data, ok := <-dataChan; ok {
			c.Render(-1, adapter.Event{Data: "data: " + data})
			return true
		}
		c.Render(-1, adapter.Event{Data: "data: [DONE]"})
		return false
	})
}

func setEventStreamHeaders(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
}

func EmbeddingProxyHandler(c *gin.Context) {
	// Retrieve the Authorization header value
	authorizationHeader := c.GetHeader("Authorization")
	// Declare a variable to store the OPENAI_API_KEY
	var openaiAPIKey string
	// Use fmt.Sscanf to extract the Bearer token
	_, err := fmt.Sscanf(authorizationHeader, "Bearer %s", &openaiAPIKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, openai.APIError{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	req := &adapter.EmbeddingRequest{}
	// Bind the JSON data from the request to the struct
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, openai.APIError{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	messages, err := req.ToGenaiMessages()
	if err != nil {
		c.JSON(http.StatusBadRequest, openai.APIError{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	ctx := c.Request.Context()
	client, err := genai.NewClient(ctx, option.WithAPIKey(openaiAPIKey))
	if err != nil {
		log.Printf("new genai client error %v\n", err)
		c.JSON(http.StatusBadRequest, openai.APIError{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}
	defer client.Close()

	model := req.ToGenaiModel()
	gemini := adapter.NewGeminiAdapter(client, model)

	resp, err := gemini.GenerateEmbedding(ctx, messages)
	if err != nil {
		log.Printf("genai generate content error %v\n", err)
		c.JSON(http.StatusBadRequest, openai.APIError{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}
