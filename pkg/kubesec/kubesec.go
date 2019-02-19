package kubesec

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/ghodss/yaml"

	"k8s.io/api/core/v1"
)

//Result represents a result returned by kubesec.io.
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

//GetKubesecPod ...
func GetKubesecPod(pod *v1.Pod) (*Result, error) {
	yamlStr, err := yaml.Marshal(*pod)
	if err != nil {
		log.Println("[kubesec] failure to marshall pod", err)
		return nil, err
	}

	return GetKubesecPodYAML(yamlStr)
}

//GetKubesecPodYAML ...
func GetKubesecPodYAML(yamlStr []byte) (*Result, error) {
	kubesecService := "https://kubesec.io/"

	multif := &bytes.Buffer{}
	writer := multipart.NewWriter(multif)
	part, err := writer.CreateFormFile("uploadfile", "object.yaml")
	if err != nil {
		log.Println("[kubesec] CreateFormFile failure", err)
		return nil, err
	}

	part.Write(yamlStr)
	writer.Close()

	req, err := http.NewRequest(http.MethodPost, kubesecService, multif)
	if err != nil {
		log.Println("[kubesec] POST  failure", err)
		return nil, err
	}
	req.Header.Add("Content-Type", writer.FormDataContentType())

	//	proxyURL, _ := url.Parse("http://127.0.0.1:8080")
	client := &http.Client{
		Timeout: time.Second * 3,
		Transport: &http.Transport{
			//			Proxy: http.ProxyURL(proxyURL),
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Println("[kubesec] client.Do failure", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println("[kubesec] read body failure", err)
			return nil, err
		}
		var k Result
		if err := json.Unmarshal(body, &k); err != nil {
			log.Println("[kubesec] Unmarshal body failure", err)
		}
		return &k, nil
		//fmt.Printf("response Body\n %v\n", k)
	}

	return nil, nil
}
