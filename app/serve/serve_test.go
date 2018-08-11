package serve

import (
	"testing"
	"context"
	"net/http"
	"io/ioutil"
)

func TestHandler_Generator(t *testing.T) {
	s:= NewServe(":8199")
	go s.Start()
	i := s.Interface()
	server := i.(GenerateServiceServer)
	req := &Request{}
	req.NodeId = 18
	response, _ := server.Generator(context.Background(), req)
	t.Log("response ", response)
	s.Close()
}

func TestServerStart(t *testing.T) {
	s:= NewServe(":8199")
	s.Start()
}

func TestHttp(t *testing.T) {
	resp, _ := http.Get("http://127.0.0.1:8199/v1/sf-gen/11?node_id=1")
	buf, _ := ioutil.ReadAll(resp.Body)
	t.Log("response", string(buf))
}

func BenchmarkGrpcHandler_Generator(b *testing.B) {
	for i := 0; i < b.N; i++ {

	}
}
