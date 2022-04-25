package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/api/public"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

const URLGetDinosaur = "http://127.0.0.1:8000/api/dinosaurs_mgmt/v1/dinosaurs"
const URLPutDinosaurStatus = "http://127.0.0.1:8000/api/dinosaurs_mgmt/v1/dinosaurs/%s/status"

func main() {
	ocmToken := os.Getenv("OCM_TOKEN")
	if ocmToken == "" {
		log.Fatal("empty ocm token")
	}

	buf := bytes.Buffer{}
	r, err := http.NewRequest(http.MethodGet, URLGetDinosaur, &buf)
	if err != nil {
		log.Fatal(err)
	}

	r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ocmToken))
	client := http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		log.Fatal(err)
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("GOT RESPONSE: %s\n\n", string(respBody))

	list := public.DinosaurRequestList{}
	err = json.Unmarshal(respBody, &list)
	if err != nil {
		log.Fatal(err)
	}

	for _, v := range list.Items {
		fmt.Println("received cluster: %s", v.Name)
		fmt.Printf("Calling to update status %q\n", fmt.Sprintf(URLPutDinosaurStatus, v.Id))
	}
}
