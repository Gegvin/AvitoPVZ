package tests

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

const baseURL = "http://localhost:8080"

// Интеграционный тест: создание ПВЗ, приёмки, добавление 50 товаров и закрытие приёмки
func TestIntegration(t *testing.T) {
	// Даем время на запуск серверов
	time.Sleep(5 * time.Second)

	// Используем dummyLogin для модератора
	token, err := getDummyToken("moderator")
	if err != nil {
		t.Fatal(err)
	}

	// Создаем ПВЗ
	pvz := map[string]string{
		"city": "Москва",
	}
	pvzBytes, _ := json.Marshal(pvz)
	req, _ := http.NewRequest("POST", baseURL+"/pvz", bytes.NewBuffer(pvzBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(res.Body)
		t.Fatalf("PVZ creation failed: %s", string(body))
	}
	var pvzResp map[string]interface{}
	json.NewDecoder(res.Body).Decode(&pvzResp)
	pvzId, ok := pvzResp["id"].(string)
	if !ok {
		t.Fatal("Invalid PVZ response")
	}

	// Получаем токен для сотрудника
	empToken, err := getDummyToken("employee")
	if err != nil {
		t.Fatal(err)
	}

	// Создаем приёмку
	receptionReq := map[string]string{"pvzId": pvzId}
	receptionBytes, _ := json.Marshal(receptionReq)
	req, _ = http.NewRequest("POST", baseURL+"/receptions", bytes.NewBuffer(receptionBytes))
	req.Header.Set("Authorization", "Bearer "+empToken)
	req.Header.Set("Content-Type", "application/json")
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(res.Body)
		t.Fatalf("Reception creation failed: %s", string(body))
	}
	res.Body.Close()

	// Добавляем 50 товаров
	for i := 0; i < 50; i++ {
		product := map[string]string{
			"type":  "электроника",
			"pvzId": pvzId,
		}
		prodBytes, _ := json.Marshal(product)
		req, _ = http.NewRequest("POST", baseURL+"/products", bytes.NewBuffer(prodBytes))
		req.Header.Set("Authorization", "Bearer "+empToken)
		req.Header.Set("Content-Type", "application/json")
		res, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != http.StatusCreated {
			body, _ := ioutil.ReadAll(res.Body)
			t.Fatalf("Product addition failed at iteration %d: %s", i, string(body))
		}
		res.Body.Close()
	}

	// Закрываем приёмку
	req, _ = http.NewRequest("POST", baseURL+"/pvz/"+pvzId+"/close_last_reception", nil)
	req.Header.Set("Authorization", "Bearer "+empToken)
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(res.Body)
		t.Fatalf("Reception close failed: %s", string(body))
	}
	res.Body.Close()
}

func getDummyToken(role string) (string, error) {
	payload := map[string]string{"role": role}
	b, _ := json.Marshal(payload)
	resp, err := http.Post(baseURL+"/dummyLogin", "application/json", bytes.NewBuffer(b))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", errors.New("dummy login failed: " + string(body))
	}
	var token string
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return "", err
	}
	return token, nil
}
