package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/textproto"
	"strings"
)

type Upstream struct {
	host string
}

type Handler struct {
	Upstreams     []*Upstream
	Transport     http.RoundTripper
	LoadBalancing LoadBalancer
}

type HandlerResponse struct {
	StatusCode int    `json:"status_code"`
	Data       any    `json:"data"`
	Message    string `json:"message"`
}

type CallInfo struct {
	Upstream   *Upstream
	Host, Port string
}

func (u *Upstream) fillCallInfo(r *http.Request) (CallInfo, error) {
	host, port, err := net.SplitHostPort(u.host)
	if err != nil {
		return CallInfo{}, err
	}

	return CallInfo{
		Upstream: u,
		Host:     host,
		Port:     port,
	}, nil
}

func Response(w http.ResponseWriter, statusCode int, data any, msg string) {
	he := HandlerResponse{
		StatusCode: statusCode,
		Data:       data,
		Message:    msg,
	}
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "application/json")
	b, _ := json.Marshal(he)
	w.Write(b)
}

func (h *Handler) RoundTrip(r *http.Request) (*http.Response, error) {
	resp, err := h.Transport.RoundTrip(r)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// CloneRequest 对 http.Request 进行深拷贝
func CloneRequest(ori *http.Request) *http.Request {
	req := ori.Clone(ori.Context())
	// TODO modify Scheme and Host
	req.URL.Scheme = ""
	req.URL.Host = ""
	return req
}

// RemoveConnectionHeader 删除 Connection 请求头
// 代理服务器转发请求时，需要删除 Connection 头，避免下游服务器误解客户端与代理之间的连接状态
// 具体可见 https://www.rfc-editor.org/rfc/rfc7230#section-6.1
func RemoveConnectionHeader(h http.Header) {
	for _, f := range h["Connection"] {
		for _, sf := range strings.Split(f, ",") {
			if sf = textproto.TrimString(sf); sf != "" {
				h.Del(sf)
			}
		}
	}
}

func UpgradeType(h http.Header) string {
	if h.Get("Upgrade") == "" {
		return ""
	}

	return strings.ToLower(h.Get("Upgrade"))
}

// addForwardedHeaders 用于添加 X-Forwarded-* 头部
// 该头部主要用于在代理/反向代理服务器中转发请求时，表示原始客户端的信息
// 常用的头包括：
// X-Forwarded-For: 记录原始客户端的 IP 地址
// X-Forwarded-Proto: 记录原始请求使用的协议，如 HTTP 或 HTTPS
// X-Forwarded-Host: 记录原始请求的 Host 头
// X-Forwarded-Port: 记录原始请求的端口
func (h *Handler) addForwardedHeaders(r *http.Request) error {
	clientIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// 避免潜在的信任来自客户端的头
		r.Header.Del("X-Forwarded-For")
		r.Header.Del("X-Forwarded-Proto")
		r.Header.Del("X-Forwarded-Host")
		return nil
	}

	proto := "https"
	if r.TLS == nil {
		proto = "http"
	}

	r.Header.Set("X-Forwarded-For", clientIP)
	r.Header.Set("X-Forwarded-Proto", proto)
	r.Header.Set("X-Forwarded-Host", r.Host)

	return nil
}

func (h *Handler) prepareRequest(r *http.Request) (*http.Request, error) {
	r = CloneRequest(r)

	r.Close = false

	// 如果客户端的请求 UA 传空，则将其强行置空防止被标准库默认赋值
	if r.Header.Get("User-Agent") == "" {
		r.Header.Set("User-Agent", "")
	}

	RemoveConnectionHeader(r.Header)

	// 支持变更协议
	upgradeType := UpgradeType(r.Header)
	if upgradeType != "" {
		r.Header.Set("Connection", "Upgrade")
		r.Header.Set("Upgrade", upgradeType)
	}

	// 添加 X-Forwarded-* 头部
	err := h.addForwardedHeaders(r)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (h *Handler) directRequest(r *http.Request, info CallInfo) {
	r.URL.Host = info.Host
}

func (h *Handler) reverseProxy(r *http.Request, callInfo CallInfo) error {
	h.directRequest(r, callInfo)

	resp, err := h.Transport.RoundTrip(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fmt.Println(resp.StatusCode)

	return nil
}

func (h *Handler) proxy(r *http.Request, or *http.Request, w http.ResponseWriter) error {

	upstream := h.LoadBalancing.Select(h.Upstreams)
	if upstream == nil {
		return errors.New("proxy error")
	}

	callInfo, err := upstream.fillCallInfo(r)
	if err != nil {
		return err
	}

	h.reverseProxy(r, callInfo)

	return nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Receive a request from ", r.RemoteAddr)
	clonedRequest, err := h.prepareRequest(r)
	if err != nil {
		Response(w, http.StatusInternalServerError, nil, fmt.Sprintf("err preparing request: %v", err))
		return
	}

	err = h.proxy(clonedRequest, r, w)
	if err != nil {
		Response(w, http.StatusInternalServerError, nil, fmt.Sprintf("err proxy: %v", err))
	}
}

var (
	addr          string
	hosts         string
	balancePolicy string
)

func init() {
	flag.StringVar(&addr, "addr", ":80", "")
	flag.StringVar(&hosts, "upstreams", "", "")
	flag.StringVar(&balancePolicy, "balance-policy", "random", "load balance policy")
}

func hostsToUpstreams(hostsStr string) []*Upstream {
	var upstreams []*Upstream
	for _, s := range strings.Split(hostsStr, " ") {
		upstreams = append(upstreams, &Upstream{host: s})
	}
	return upstreams
}

func main() {
	flag.Parse()
	if hosts == "" {
		log.Fatal("upstream is required")
	}

	upstreams := hostsToUpstreams(hosts)

	handler := &Handler{
		Upstreams: upstreams,
		Transport: http.DefaultTransport,
	}

	switch balancePolicy {
	case "random":
		handler.LoadBalancing = &RandomSelector{}
	case "robin":
		handler.LoadBalancing = &RoundRobinSelector{}
	}

	log.Println(http.ListenAndServe(addr, handler))
}
