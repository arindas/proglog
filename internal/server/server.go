package server

import (
	"encoding/json"
	"log"
	"net/http"
)

type httpServer struct {
	log *Log
}

func newHttpServer() *httpServer {
	return &httpServer{log: NewLog()}
}

type ProduceRequest struct {
	Record Record `json:"record"`
}

type ProduceResponse struct {
	Offset uint64 `json:"offset"`
}

type ConsumeRequest struct {
	Offset uint64 `json:"offset"`
}

type ConsumeResponse struct {
	Record Record `json:"record"`
}

type safeRequest struct {
	respWriter http.ResponseWriter
	request    *http.Request

	err           error
	errSource     string
	httpErrorCode int
}

func newSafeRequest(rw http.ResponseWriter, r *http.Request) safeRequest {
	return safeRequest{respWriter: rw, request: r}
}

func (req *safeRequest) decodeJsonBodyTo(val interface{}) {
	if req.err != nil {
		return
	}

	decoder := json.NewDecoder(req.request.Body)
	decoder.DisallowUnknownFields()

	req.err = decoder.Decode(val)
	req.errSource = "safeRequest.decodeJsonBodyTo"
	req.httpErrorCode = http.StatusBadRequest
}

func (req *safeRequest) respondWithJson(val interface{}) {
	if req.err != nil {
		return
	}

	encoder := json.NewEncoder(req.respWriter)
	req.err = encoder.Encode(val)
	req.errSource = "safeRequest.respondWithJson"
	req.httpErrorCode = http.StatusInternalServerError
}

func (req *safeRequest) emitErrors() {
	if req.err != nil {
		http.Error(req.respWriter, req.err.Error(), req.httpErrorCode)

		log.Printf("error(%v) at [%v]", req.err, req.errSource)
	}
}

type safeProduceRequest struct {
	safeRequest

	produceReq ProduceRequest

	offset uint64
}

func (req *safeProduceRequest) parse() { req.decodeJsonBodyTo(&req.produceReq) }

func (req *safeProduceRequest) service(log *Log) {
	if req.err != nil {
		return
	}

	req.offset, req.err = log.Append(req.produceReq.Record)
	req.errSource = "safeProduceRequest.service"
	req.httpErrorCode = http.StatusInternalServerError
}

func (req *safeProduceRequest) respond() {
	req.respondWithJson(ProduceResponse{Offset: req.offset})
	req.emitErrors()
}

func (serv *httpServer) handleProduce(rw http.ResponseWriter, r *http.Request) {
	req := safeProduceRequest{safeRequest: newSafeRequest(rw, r)}

	req.parse()
	req.service(serv.log)
	req.respond()
}

type safeConsumeRequest struct {
	safeRequest

	consumeReq ConsumeRequest

	record Record
}

func (req *safeConsumeRequest) parse() { req.decodeJsonBodyTo(&req.consumeReq) }

func (req *safeConsumeRequest) service(log *Log) {
	if req.err != nil {
		return
	}

	req.record, req.err = log.Read(req.consumeReq.Offset)
	req.errSource = "safeConsumeRequest.service"
	req.httpErrorCode = http.StatusNotFound
}

func (req *safeConsumeRequest) respond() {
	req.respondWithJson(ConsumeResponse{Record: req.record})
	req.emitErrors()
}

func (serv *httpServer) handleConsume(rw http.ResponseWriter, r *http.Request) {
	req := safeConsumeRequest{safeRequest: newSafeRequest(rw, r)}

	req.parse()
	req.service(serv.log)
	req.respond()
}
