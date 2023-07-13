package altc

import (
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MicahParks/keyfunc/v2"
	"github.com/gogama/httpx"
	"github.com/gogama/httpx/request"
	"github.com/golang-jwt/jwt/v5"
	"io"
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
	sendTimeout         = 30 * time.Second
	authUriEnv          = "AUTH_URI"
	authClientIdEnv     = "AUTH_CLIENT_ID"
	authSecretEnv       = "AUTH_SECRET"
	authPublicKeySetEnv = "AUTH_PUBLIC_KEY_SET"
	authAudienceEnv     = "AUTH_AUDIENCE"
	authIssuerEnv       = "AUTH_ISSUER"
)

func NewClient() *Client {
	return &Client{}
}

func (c *Client) Register(ctx context.Context) error {

	authTokenId, err := c.getAuthToken()
	if err != nil {
		fmt.Println(err)
		return err
	}

	fmt.Println("authTokenId:", authTokenId)
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

		execution, err := send(clusterResources)
		if err != nil {
			fmt.Println(fmt.Sprintf("error sending resources on attempt %d: %s", attempts, err.Error()))
			// Don't return the error from the conditionFunc, doing so will abort the retry.
			// The point of the retry is to not consider an error an actual error if the condition
			// succeeds before the max retry.
			// 'done' is false since the condition has not succeeded yet
			return false, nil
		}
		if execution.Response.StatusCode != 200 {
			fmt.Println(fmt.Sprintf("response from altc-nodeserver (%d): %s", execution.Response.StatusCode, execution.Response.Body))
		}

		return true, nil
	})

	return err
}

func send(clusterResources *ClusterResources) (*request.Execution, error) {
	fmt.Println(fmt.Sprintf("sending %d clusterResources items", len((*clusterResources).Data)))
	client := &httpx.Client{}
	pr, pw := io.Pipe()

	defer pr.Close()

	/* Write to the pipe asynchronously because the pipe uses channels to synchronize
	   writing to the pipe with reading from the pipe: there must be a reader waiting
	   to receive the bytes written before data is written to the pipe.
	   This is because after writing to the pipe (after sending the bytes to write to
	   the write channel), the writer waits for the number of bytes read to be sent
	   to the read channel, which happens during the read operation after the bytes
	   are received from write channel.
	*/
	go func() {
		gw := gzip.NewWriter(pw)

		if err := json.NewEncoder(gw).Encode(clusterResources); err != nil {
			fmt.Println("error encoding gzip data:", err)
		}

		if err := gw.Close(); err != nil {
			fmt.Println("error closing gzip writer:", err)
		}
		defer func() {
			if err := pw.Close(); err != nil {
				fmt.Println("error closing pipe writer:", err)
			}
		}()
	}()

	plan, err := request.NewPlan("POST", "http://altc-nodeserver:8080/kubernetes/resource", pr)
	if err != nil {
		return nil, err
	}
	plan.Header.Set("Content-Type", "application/json")
	plan.Header.Set("Content-Encoding", "gzip")

	return client.Do(plan)
}

func (c *Client) getAuthToken() (string, error) {

	authUri := os.Getenv(authUriEnv)
	authClientId := os.Getenv(authClientIdEnv)
	authSecret := os.Getenv(authSecretEnv)
	authPublicKeySetBytes, _ := base64.StdEncoding.DecodeString(os.Getenv(authPublicKeySetEnv))
	authIssuerBytes, _ := base64.StdEncoding.DecodeString(os.Getenv(authIssuerEnv))
	authIssuer := string(authIssuerBytes)
	authAudienceBytes, _ := base64.StdEncoding.DecodeString(os.Getenv(authAudienceEnv))
	authAudience := string(authAudienceBytes)

	payloadObj := AuthPayload{
		ClientId:     authClientId,
		ClientSecret: authSecret,
		Audience:     authAudience,
		GrantType:    "client_credentials",
	}

	payloadBytes, err := json.Marshal(payloadObj)
	if err != nil {
		return "", errors.New(fmt.Sprintf("Error marshalling payload: %s", err))
	}

	fmt.Println("Authorizing...")
	payload := strings.NewReader(string(payloadBytes))
	req, err := http.NewRequest("POST", authUri, payload)
	if err != nil {
		return "", errors.New(fmt.Sprintf("error creating request: %s", err))
	}

	req.Header.Add("content-type", "application/json")
	res, err := http.DefaultClient.Do(req)
	defer res.Body.Close()

	if err != nil {
		return "", errors.New(fmt.Sprintf("authorization request error: %s", err))
	}

	authResponseBody, err := io.ReadAll(res.Body)
	type AuthResponse struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}

	authResponse := AuthResponse{}
	err = json.Unmarshal(authResponseBody, &authResponse)
	if err != nil {
		return "", errors.New(fmt.Sprintf("error unmarshalling authResponseString: %s", err))
	}

	rawAuthPublicKeySet := json.RawMessage(authPublicKeySetBytes)
	jwks, err := keyfunc.NewJSON(rawAuthPublicKeySet)
	if err != nil {
		return "", errors.New(fmt.Sprintf("error creating jwks: %s", err))
	}

	type CustomClaims struct {
		TokenId string `json:"https://altconsole.register.com/clientTokenId"`
		jwt.RegisteredClaims
	}
	claims := &CustomClaims{}

	_, err = jwt.ParseWithClaims(authResponse.AccessToken, claims, jwks.Keyfunc)
	if err != nil {
		return "", errors.New(fmt.Sprintf("error processing jwt claims: %s", err))
	}

	// Validate issuer and audience
	if claims.RegisteredClaims.Issuer != authIssuer {
		return "", errors.New(fmt.Sprintf("unexpected issuer: %s", claims.RegisteredClaims.Issuer))
	}

	if claims.RegisteredClaims.Audience[0] != authAudience {
		return "", errors.New(fmt.Sprintf("unexpected issuer: %s", claims.RegisteredClaims.Audience[0]))
	}

	/*
		fmt.Println("issuer  :", claims.RegisteredClaims.Issuer)
		fmt.Println("audience:", claims.RegisteredClaims.Audience[0])
		fmt.Println("expires :", claims.RegisteredClaims.ExpiresAt)
	*/

	return claims.TokenId, nil
}
