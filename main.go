package main

import (
	"github.com/valyala/fasthttp"
	"log"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"bytes"
	"net/url"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client/metadata"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/signer/v4"
	"flag"
)

var (
	creds *credentials.Credentials
	port int
)

func ReverseProxyHandler(ctx *fasthttp.RequestCtx) {
	req := &ctx.Request
	resp := &ctx.Response
	prepareRequest(req)
	if err := fasthttp.Do(req, resp); err != nil {
		ctx.Logger().Printf("error when proxying the request: %s", err)
	}
	postprocessResponse(resp)
}

func prepareRequest(req *fasthttp.Request) {
	req.Header.Del("Connection")
	auth := req.Header.Peek("Authorization")
	if len(auth) != 0 {
		transformRequest(req, auth)
		log.Println(req)
	}
}

func postprocessResponse(resp *fasthttp.Response) {
	resp.Header.Del("Connection")
}

func transformRequest(req *fasthttp.Request, auth []byte) {
	endpoint, region, service := resolve(auth)
	fmt.Println(endpoint.Host)
	req.URI().SetScheme(endpoint.Scheme)
	req.SetHost(endpoint.Host)
	config := aws.NewConfig().WithCredentials(creds).WithRegion(region)
	clientInfo := metadata.ClientInfo{
		ServiceName: service,
	}
	awsRequestFormat(req, endpoint, config, clientInfo)
}

func awsRequestFormat(req *fasthttp.Request, endpoint *url.URL, config *aws.Config, clientInfo metadata.ClientInfo) {
	operation := &request.Operation{
		Name:       "",
		HTTPMethod: string(req.Header.Method()),
		HTTPPath:   string(req.URI().Path()),
	}
	handlers := request.Handlers{}
	handlers.Sign.PushBack(v4.SignSDKRequest)
	awsReq := request.New(*config, clientInfo, handlers, nil, operation, nil, nil)
	headerFormat(req, awsReq)
}

func headerFormat(req *fasthttp.Request, awsReq *request.Request) {
	buf := req.Body()
	awsReq.SetBufferBody(buf)
	uri := req.URI()
	awsReq.HTTPRequest.URL = &url.URL{
		Scheme:     string(uri.Scheme()),
		Host:       string(uri.Host()),
		Path:       string(uri.Path()),
		RawQuery:   string(uri.QueryString()),
		RawPath:    string(uri.PathOriginal()),
		ForceQuery: false,
	}
	awsReq.HTTPRequest.Header.Set("Content-Md5", string(req.Header.Peek("Content-Md5")))
	if err := awsReq.Sign(); err != nil {
		log.Printf("error signing: %v\n", err)
	}
	for k, v := range awsReq.HTTPRequest.Header {
		req.Header.Set(k, v[0])
	}
}

func resolve(auth []byte) (endpoint *url.URL, region string, service string) {
	s := bytes.Split(auth, []byte(" "))
	info := bytes.Split(bytes.Split(s[1], []byte("="))[1], []byte("/"))
	region, service = string(info[2]), string(info[3])
	resolver := endpoints.DefaultResolver()
	e, _ := resolver.EndpointFor(service, region)
	endpoint, _ = url.Parse(e.URL)
	return
}

func init() {
	flag.IntVar(&port, "port", 8082, "Binding port")
}

func main() {
	creds = defaults.CredChain(defaults.Config(), defaults.Handlers())
	if _, err := creds.Get(); err != nil {
		fmt.Println(err)
		return
	}
	if err := fasthttp.ListenAndServe(fmt.Sprintf(":%d", port), ReverseProxyHandler); err != nil {
		log.Println("error in fasthttp server: %s", err)
	}
}
