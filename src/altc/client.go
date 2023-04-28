package altc

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gogama/httpx"
	"io"

	//"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/wait"
	"net/http"
	"os"
	"strings"
	"time"
)

type Client struct {
}

type AuthPayload struct {
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Audience     string `json:"audience"`
	GrantType    string `json:"grant_type"`
}

const (
	sendTimeout     = 30 * time.Second
	authUriEnv      = "AUTH_URI"
	authClientIdEnv = "AUTH_CLIENT_ID"
	authSecretEnv   = "AUTH_SECRET"
	audience        = "https://altconsole.register.com"
)

func NewClient() *Client {
	return &Client{}
}

func (c *Client) Register(ctx context.Context) error {

	authUri := os.Getenv(authUriEnv)
	authClientId := os.Getenv(authClientIdEnv)
	authSecret := os.Getenv(authSecretEnv)

	payloadObj := AuthPayload{
		ClientId:     authClientId,
		ClientSecret: authSecret,
		Audience:     audience,
		GrantType:    "client_credentials",
	}
	payloadBytes, err := json.Marshal(payloadObj)
	if err != nil {
		fmt.Println("Error marshalling payload", err)
		return err
	}

	payload := strings.NewReader(string(payloadBytes))
	req, err := http.NewRequest("POST", authUri, payload)
	if err != nil {
		return err
	}

	req.Header.Add("content-type", "application/json")

	res, err := http.DefaultClient.Do(req)
	defer res.Body.Close()

	if err != nil {
		return err
	}

	fmt.Println("response from authorization:")
	fmt.Println(res.StatusCode)
	body, err := io.ReadAll(res.Body)
	fmt.Println(string(body))
	return nil
}

func (c *Client) Send(ctx context.Context, clusterResources *ClusterResources) error {

	ctx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()
	maxSteps := 4

	backoff := wait.Backoff{
		Duration: 500 * time.Millisecond,
		Factor:   2,
		Jitter:   0.0,
		Steps:    maxSteps,
	}

	attempts := 0
	err := wait.ExponentialBackoffWithContext(ctx, backoff, func() (done bool, err error) {
		attempts++
		fmt.Println("attempt", attempts)

		clusterResourcesJson, err := json.Marshal(*clusterResources)
		if err != nil {
			fmt.Println(fmt.Sprintf("ERROR: error marshalling clusterResources: %s", err))
			// Don't return the error from the conditionFunc, doing so will abort the retry
			return false, nil
		}
		fmt.Println(fmt.Sprintf("sending %d clusterResources items", len((*clusterResources).Data)))
		client := &httpx.Client{}
		resp, err := client.Post("http://altc-nodeserver:8080/kubernetes/resource", "application/json", clusterResourcesJson)
		if err != nil {
			fmt.Println(fmt.Sprintf("error sending resources on attempt %d: %s", attempts, err.Error()))
			// Don't return the error from the conditionFunc, doing so will abort the retry.
			// The point of the retry is to not consider an error an actual error if the condition
			// succeeds before the max retry.
			// 'done' is false since the condition has not succeeded yet
			return false, nil
		}
		fmt.Println(fmt.Sprintf("response from altc-nodeserver (%d): %s", resp.StatusCode(), string(resp.Body)))
		return true, nil
	})

	return err
}
