package tests

import (
	"bytes"
	"encoding/json"

	"io"
	"net/http"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

const baseURL = "http://localhost:8080"

func getDummyToken(t *testing.T, role string) string {
	t.Helper()
	payload := map[string]string{"role": role}
	b, err := json.Marshal(payload)
	require.NoError(t, err, "Failed to marshal dummy login payload")

	resp, err := http.Post(baseURL+"/dummyLogin", "application/json", bytes.NewBuffer(b))
	require.NoError(t, err, "Failed to make dummy login request")
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read dummy login response body")

	require.Equal(t, http.StatusOK, resp.StatusCode, "dummy login failed: %s", string(bodyBytes))

	var tokenResp struct {
		Token *string `json:"token"`
	}

	err = json.Unmarshal(bodyBytes, &tokenResp)
	require.NoError(t, err, "Failed to decode token response JSON: %s", string(bodyBytes))
	require.NotNil(t, tokenResp.Token, "Token field was nil in dummy login response")
	require.NotEmpty(t, *tokenResp.Token, "Token field was empty in dummy login response")

	return *tokenResp.Token
}

func TestIntegration_PVZ_Reception_Products(t *testing.T) {

	t.Log("Waiting for services to start...")
	time.Sleep(10 * time.Second)
	t.Log("Attempting integration test...")

	t.Log("Getting moderator token...")
	modToken := getDummyToken(t, "moderator")
	t.Log("Moderator token obtained.")

	pvzCity := "Москва"
	pvzPayload := map[string]string{"city": pvzCity}
	pvzBytes, err := json.Marshal(pvzPayload)
	require.NoError(t, err, "Failed to marshal PVZ payload")

	req, err := http.NewRequest("POST", baseURL+"/pvz", bytes.NewBuffer(pvzBytes))
	require.NoError(t, err, "Failed to create PVZ request")
	req.Header.Set("Authorization", "Bearer "+modToken)
	req.Header.Set("Content-Type", "application/json")

	t.Logf("Attempting to create PVZ in city: %s", pvzCity)
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "Failed to execute PVZ creation request")
	defer res.Body.Close()

	pvzBodyBytes, err := io.ReadAll(res.Body)
	require.NoError(t, err, "Failed to read PVZ creation response body")
	require.Equal(t, http.StatusCreated, res.StatusCode, "PVZ creation failed: %s", string(pvzBodyBytes))

	var pvzResp map[string]interface{}
	err = json.Unmarshal(pvzBodyBytes, &pvzResp)
	require.NoError(t, err, "Failed to decode PVZ response JSON: %s", string(pvzBodyBytes))

	pvzId, ok := pvzResp["id"].(string)
	require.True(t, ok, "Invalid PVZ ID type in response")
	require.NotEmpty(t, pvzId, "PVZ ID is empty in response")
	require.Equal(t, pvzCity, pvzResp["city"], "PVZ response city name mismatch")
	t.Logf("Created PVZ ID: %s in City: %s", pvzId, pvzCity)

	t.Log("Getting employee token...")
	empToken := getDummyToken(t, "employee")
	t.Log("Employee token obtained.")

	receptionPayload := map[string]string{"pvzId": pvzId}
	receptionBytes, err := json.Marshal(receptionPayload)
	require.NoError(t, err, "Failed to marshal reception payload")

	req, err = http.NewRequest("POST", baseURL+"/receptions", bytes.NewBuffer(receptionBytes))
	require.NoError(t, err, "Failed to create reception request")
	req.Header.Set("Authorization", "Bearer "+empToken)
	req.Header.Set("Content-Type", "application/json")

	t.Logf("Attempting to create reception for PVZ ID: %s", pvzId)
	res, err = http.DefaultClient.Do(req)
	require.NoError(t, err, "Failed to execute reception creation request")
	defer res.Body.Close()

	recBodyBytes, err := io.ReadAll(res.Body)
	require.NoError(t, err, "Failed to read reception creation response body")
	require.Equal(t, http.StatusCreated, res.StatusCode, "Reception creation failed: %s", string(recBodyBytes))
	t.Log("Created Reception successfully.")

	productType := "электроника"
	t.Logf("Attempting to add %d products of type '%s'...", 50, productType)
	for i := 0; i < 50; i++ {
		productPayload := map[string]string{
			"type":  productType,
			"pvzId": pvzId,
		}
		prodBytes, err := json.Marshal(productPayload)
		require.NoError(t, err, "Failed to marshal product payload (iteration %d)", i)

		req, err = http.NewRequest("POST", baseURL+"/products", bytes.NewBuffer(prodBytes))
		require.NoError(t, err, "Failed to create product request (iteration %d)", i)
		req.Header.Set("Authorization", "Bearer "+empToken)
		req.Header.Set("Content-Type", "application/json")

		res, err = http.DefaultClient.Do(req)
		require.NoError(t, err, "Failed to execute product creation request (iteration %d)", i)

		prodBodyBytes, errRead := io.ReadAll(res.Body)
		require.NoError(t, errRead, "Failed to read product creation response body (iteration %d)", i)
		res.Body.Close()

		require.Equal(t, http.StatusCreated, res.StatusCode, "Product addition failed at iteration %d: %s", i, string(prodBodyBytes))

		if (i+1)%10 == 0 {
			t.Logf("Added product %d/50", i+1)
		}
	}
	t.Logf("Added %d Products successfully.", 50)

	t.Logf("Attempting to close reception for PVZ ID: %s", pvzId)
	req, err = http.NewRequest("POST", baseURL+"/pvz/"+pvzId+"/close_last_reception", nil)
	require.NoError(t, err, "Failed to create close reception request")
	req.Header.Set("Authorization", "Bearer "+empToken)

	res, err = http.DefaultClient.Do(req)
	require.NoError(t, err, "Failed to execute close reception request")
	defer res.Body.Close()

	closeBodyBytes, err := io.ReadAll(res.Body)
	require.NoError(t, err, "Failed to read close reception response body")
	require.Equal(t, http.StatusOK, res.StatusCode, "Reception close failed: %s", string(closeBodyBytes))
	t.Log("Closed Reception successfully.")
	t.Log("Integration test completed successfully!")
}
