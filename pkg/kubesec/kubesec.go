package kubesec

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/ghodss/yaml"
	"github.com/vasu1124/introspect/pkg/logger"

	v1 "k8s.io/api/core/v1"
)

// Result represents a result returned by kubesec.io.
type Result struct {
	Error   string `json:"error"`
	Score   int    `json:"score"`
	Scoring struct {
		Critical []struct {
			Selector string `json:"selector"`
			Reason   string `json:"reason"`
			Weight   int    `json:"weight"`
		} `json:"critical"`
		Advise []struct {
			Selector string `json:"selector"`
			Reason   string `json:"reason"`
			Href     string `json:"href,omitempty"`
		} `json:"advise"`
	} `json:"scoring"`
}

// GetKubesecPod ...
func GetKubesecPod(pod *v1.Pod) (*Result, error) {
	yamlStr, err := yaml.Marshal(*pod)
	if err != nil {
		logger.Log.Error(err, "[kubesec] failure to marshall pod")
		return nil, err
	}

	return GetKubesecPodYAML(yamlStr)
}

// GetKubesecPodYAML ...
func GetKubesecPodYAML(yamlStr []byte) (*Result, error) {
	kubesecService := "https://v2.kubesec.io/scan"

	multif := &bytes.Buffer{}
	writer := multipart.NewWriter(multif)
	part, err := writer.CreateFormFile("uploadfile", "object.yaml")
	if err != nil {
		logger.Log.Error(err, "[kubesec] CreateFormFile failure")
		return nil, err
	}

	part.Write(yamlStr)
	writer.Close()

	req, err := http.NewRequest(http.MethodPost, kubesecService, multif)
	if err != nil {
		logger.Log.Error(err, "[kubesec] POST failure")
		return nil, err
	}
	req.Header.Add("Content-Type", writer.FormDataContentType())

	//	proxyURL, _ := url.Parse("http://127.0.0.1:8080")
	client := &http.Client{
		Timeout: time.Second * 3,
		Transport: &http.Transport{
			//Proxy: http.ProxyURL(proxyURL),
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
			},
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.Log.Error(err, "[kubesec] client.Do failure")
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			logger.Log.Error(err, "[kubesec] read body failure")
			return nil, err
		}
		var k Result
		if err := json.Unmarshal(body, &k); err != nil {
			logger.Log.Error(err, "[kubesec] Unmarshal body failure")
		}
		return &k, nil
		//fmt.Printf("response Body\n %v\n", k)
	}

	return nil, nil
}
