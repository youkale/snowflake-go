package serve

import (
	"context"
	"google.golang.org/grpc"
	"snowflake"
	"google.golang.org/grpc/reflection"
	"log"
	"cmux"
	"net/http"
	"net"
	"io"
	"bytes"
	"errors"
	"strings"
	"strconv"
	"fmt"
)

type Server struct {
	addr        string
	grpcServer  *grpc.Server
	httpServer  *http.Server
	redisServer *redisServer
	grpc        *grpcHandler
	httpHandler *httpHandler
}

type grpcHandler struct {
}

type httpHandler struct {
}

const errHttpResponse = `{"status":-1,"message":"error node_id"}`
const okHttpResponse = `{"status":0,"message":"ok","data":{"id":"%s","base32":"%s","base58":"%s"}}`
const maxNode = 255

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	nodeId, err := strconv.ParseInt(r.URL.Query().Get("node_id"), 10, 64)
	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "application/json")
	if nil != err {
		w.Write([]byte(errHttpResponse))
	} else {
		ctx := context.Background()
		id, e := generator(ctx, nodeId)
		if nil != e {
			w.Write([]byte(errHttpResponse))
		} else {
			respString := fmt.Sprintf(okHttpResponse, id.String(), id.Base32(), id.Base58())
			w.Write([]byte(respString))
		}
	}

}

func (s *grpcHandler) Generator(ctx context.Context, req *Request) (*Response, error) {
	id, e := generator(ctx, req.NodeId)
	if nil != e {
		return nil, e
	}
	resp := &Response{}
	resp.Base32 = id.Base32()
	resp.Base58 = id.Base58()
	resp.Id = id.String()
	return resp, nil
}

func generator(ctx context.Context, nodeId int64) (snowflake.ID, error) {
	_, cancel := context.WithCancel(ctx)
	ni := nodeId
	if ni > maxNode {
		ni = maxNode
	}
	if ni <= 0 {
		ni = 1
	}

	node, e := snowflake.NewNode(ni)
	if nil != e {
		cancel()
		return -1, e
	}
	return node.Generate(), nil
}

func NewServe(addr string) (*Server) {
	s := &Server{}
	s.addr = addr
	s.grpcServer = grpc.NewServer()
	s.grpc = &grpcHandler{}
	s.httpHandler = &httpHandler{}
	return s
}

func (s *Server) Interface() interface{} {
	return s.grpc
}

func (s *Server) Start() error {

	l, e := net.Listen("tcp", s.addr)
	if nil != e {
		log.Fatal(e)
	}
	m := cmux.New(l)
	grpcL := m.Match(cmux.HTTP2HeaderField("content-type", "application/grpc"))
	httpL := m.Match(cmux.HTTP1Fast())
	redisL := m.Match(redisMatcher)

	reflection.Register(s.grpcServer)
	RegisterGenerateServiceServer(s.grpcServer, s.grpc)

	s.httpServer = &http.Server{
		Handler: s.httpHandler,
	}
	s.redisServer = redisServe()

	go s.redisServer.Serve(redisL)
	go s.grpcServer.Serve(grpcL)
	go s.httpServer.Serve(httpL)

	return m.Serve()

}

func (s *Server) Close() {
	if nil != s.grpcServer {
		s.grpcServer.GracefulStop()
	}
	if nil != s.httpServer {
		s.httpServer.Close()
	}
	if nil != s.redisServer {
		s.redisServer.Close()
	}
}

const (
	MessageError  = '-'
	MessageStatus = '+'
	MessageInt    = ':'
	MessageBulk   = '$'
	MessageMutli  = '*'
)

const respJson = `{"id":"%s","base32":"%s","base58":"%s"}`
const errCmd = "-ERR not support cmd"
const errNodeId = "-ERR node_id must be int"
const unSupportCmd = "-ERR not support cmd"
const snowflakeCmd = "sfid"

var CRLF = []byte{0x0d, 0x0a}

type redisServer struct {
	ctx    context.Context
	conns  chan net.Conn
	listen net.Listener
	cancel context.CancelFunc
}

func redisServe() (*redisServer) {
	s := redisServer{}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.conns = make(chan net.Conn, 16)
	return &s
}

func (r *redisServer) handler(c net.Conn) {
	defer func() {
		c.Close()
	}()

	ctx, cancel := context.WithCancel(r.ctx)
	b := make([]byte, 1024)
	n, err := c.Read(b)
	if nil != err && err == io.EOF {
		cancel()
	}
	var resp string
	split := strings.Split(string(b[:n]), string(CRLF))
	cmd := strings.ToLower(split[2])
	size, e := strconv.Atoi(split[0][1:])
	if nil != e {
		resp = errCmd
	}
	if cmd == "ping" {
		resp = "pong"
	} else if cmd == snowflakeCmd && size == 2 {
		nodeId := strings.ToLower(split[4])
		nodeNum, e := strconv.ParseInt(nodeId, 10, 64)
		if nil != e {
			resp = errNodeId
		} else {
			id, e := generator(ctx, nodeNum)
			if nil != e {
				resp = errCmd
			}
			resp = fmt.Sprintf(respJson, id.String(), id.Base32(), id.Base58())
		}

	} else {
		resp = unSupportCmd
	}
	if resp[0] != MessageError {
		resp = string(MessageStatus) + resp
	}
	resp += string(CRLF)
	c.Write([]byte(resp))
}

func (r *redisServer) Serve(lis net.Listener) {
	r.listen = lis
	go r.accept()
	go r.handleConnection()
}

func (r *redisServer) Close() {
	r.cancel()
}

func (r *redisServer) accept() error {
	for {
		select {
		case <-r.ctx.Done():
			return nil
		default:
		}
		conn, err := r.listen.Accept()
		if nil != err {
			log.Print(errors.New("failed to accepted raw connections"))
			continue
		}
		r.conns <- conn
	}
}

func (r *redisServer) handleConnection() {
	for {
		select {
		case <-r.ctx.Done():
			r.listen.Close()
		L:
			for {
				select {
				case conn := <-r.conns:
					conn.Close()
				default:
					break L
				}
			}
			return
		case conn := <-r.conns:
			go r.handler(conn)
		}
	}
}

func redisMatcher(r io.Reader) bool {
	b := make([]byte, 4)
	_, err := r.Read(b)
	if err == io.EOF {
		return false
	}
	switch b[0] {
	case MessageError, MessageStatus, MessageInt, MessageBulk, MessageMutli:
		break
	default:
		return false
	}
	if b[1] < 0x30 || b[1] > 0x39 {
		return false
	}
	return bytes.Equal(b[2:4], CRLF)
}
