package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type loginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	UserID      string `json:"user_id"`
	TenantID    string `json:"tenant_id"`
	Role        string `json:"role"`
}

type ticketResponse struct {
	ID              string  `json:"id"`
	TenantID        string  `json:"tenant_id"`
	Subject         string  `json:"subject"`
	Status          string  `json:"status"`
	CreatedByUserID string  `json:"created_by_user_id"`
	AssignedToUser  *string `json:"assigned_to_user_id"`
}

type documentResponse struct {
	ID              string `json:"id"`
	TenantID        string `json:"tenant_id"`
	Title           string `json:"title"`
	SourceURI       string `json:"source_uri"`
	CreatedByUserID string `json:"created_by_user_id"`
	CreatedAt       string `json:"created_at"`
}

var _ = Describe("API black-box scenarios", Ordered, func() {
	const (
		seedTenantID = "22222222-2222-2222-2222-222222222222"
		seedEmail    = "admin+local@example.com"
		seedPassword = "password"
	)

	var (
		baseURL    string
		httpClient *http.Client
		token      string
	)

	BeforeAll(func() {
		baseURL = strings.TrimRight(getEnv("E2E_BASE_URL", "http://localhost:18080"), "/")
		httpClient = &http.Client{Timeout: 10 * time.Second}

		Eventually(func(g Gomega) {
			statusCode, body, err := doRequest(httpClient, http.MethodGet, baseURL+"/healthz", "", nil)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(statusCode).To(Equal(http.StatusOK), "healthz body: %s", string(body))
		}, 2*time.Minute, 2*time.Second).Should(Succeed())
	})

	It("rejects unauthorized request on protected endpoint", func() {
		statusCode, _, err := doRequest(httpClient, http.MethodGet, baseURL+"/v1/tickets", "", nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(statusCode).To(Equal(http.StatusUnauthorized))
	})

	It("logs in with seeded credentials and returns bearer token", func() {
		payload := map[string]any{
			"email":     seedEmail,
			"password":  seedPassword,
			"tenant_id": seedTenantID,
		}

		statusCode, body, err := doRequest(httpClient, http.MethodPost, baseURL+"/v1/auth/login", "", payload)
		Expect(err).NotTo(HaveOccurred())
		Expect(statusCode).To(Equal(http.StatusOK), "login body: %s", string(body))

		var response loginResponse
		Expect(json.Unmarshal(body, &response)).To(Succeed())
		Expect(response.AccessToken).NotTo(BeEmpty())
		Expect(response.TokenType).To(Equal("Bearer"))
		Expect(response.UserID).NotTo(BeEmpty())
		Expect(response.TenantID).To(Equal(seedTenantID))
		Expect(response.Role).To(Equal("admin"))

		token = response.AccessToken
	})

	It("creates, reads, updates and lists tickets", func() {
		Expect(token).NotTo(BeEmpty())
		uniqueSubject := fmt.Sprintf("E2E ticket %d", time.Now().UnixNano())

		statusCode, body, err := doRequest(httpClient, http.MethodPost, baseURL+"/v1/tickets", token, map[string]any{
			"subject": uniqueSubject,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(statusCode).To(Equal(http.StatusCreated), "create ticket body: %s", string(body))

		var created ticketResponse
		Expect(json.Unmarshal(body, &created)).To(Succeed())
		Expect(created.Subject).To(Equal(uniqueSubject))
		Expect(created.Status).To(Equal("open"))
		Expect(uuid.Validate(created.ID)).To(Succeed())

		statusCode, body, err = doRequest(httpClient, http.MethodGet, baseURL+"/v1/tickets", token, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(statusCode).To(Equal(http.StatusOK), "list tickets body: %s", string(body))

		var tickets []ticketResponse
		Expect(json.Unmarshal(body, &tickets)).To(Succeed())
		Expect(tickets).To(ContainElement(HaveField("ID", created.ID)))

		statusCode, body, err = doRequest(httpClient, http.MethodGet, baseURL+"/v1/tickets/"+created.ID, token, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(statusCode).To(Equal(http.StatusOK), "get ticket body: %s", string(body))

		var fetched ticketResponse
		Expect(json.Unmarshal(body, &fetched)).To(Succeed())
		Expect(fetched.ID).To(Equal(created.ID))
		Expect(fetched.Subject).To(Equal(uniqueSubject))

		updatedSubject := uniqueSubject + " (updated)"
		statusCode, body, err = doRequest(httpClient, http.MethodPatch, baseURL+"/v1/tickets/"+created.ID, token, map[string]any{
			"subject": updatedSubject,
			"status":  "closed",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(statusCode).To(Equal(http.StatusOK), "update ticket body: %s", string(body))

		var updated ticketResponse
		Expect(json.Unmarshal(body, &updated)).To(Succeed())
		Expect(updated.ID).To(Equal(created.ID))
		Expect(updated.Subject).To(Equal(updatedSubject))
		Expect(updated.Status).To(Equal("closed"))
	})

	It("ingests and lists knowledge documents", func() {
		Expect(token).NotTo(BeEmpty())
		uniqueTitle := fmt.Sprintf("E2E doc %d", time.Now().UnixNano())

		statusCode, body, err := doRequest(httpClient, http.MethodPost, baseURL+"/v1/documents", token, map[string]any{
			"title":      uniqueTitle,
			"source_uri": "https://example.com/e2e-doc",
			"content":    "This is a black-box e2e document ingestion scenario.",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(statusCode).To(Equal(http.StatusCreated), "ingest document body: %s", string(body))

		var created documentResponse
		Expect(json.Unmarshal(body, &created)).To(Succeed())
		Expect(created.Title).To(Equal(uniqueTitle))
		Expect(uuid.Validate(created.ID)).To(Succeed())

		statusCode, body, err = doRequest(httpClient, http.MethodGet, baseURL+"/v1/documents", token, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(statusCode).To(Equal(http.StatusOK), "list documents body: %s", string(body))

		var documents []documentResponse
		Expect(json.Unmarshal(body, &documents)).To(Succeed())
		Expect(documents).To(ContainElement(HaveField("ID", created.ID)))
	})
})

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func doRequest(client *http.Client, method, url, token string, body any) (int, []byte, error) {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return 0, nil, err
	}

	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}

	return resp.StatusCode, responseBody, nil
}
