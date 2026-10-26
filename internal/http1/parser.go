package http1

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func ParseRequest(reader *bufio.Reader) (*http.Request, error) {

	req := http.Request{}

	requestLine, err := reader.ReadString('\n')
	if err != nil {
		return nil, io.EOF
	}
	requestLine = strings.TrimSuffix(requestLine, "\r\n")
	split := strings.Split(requestLine, " ")
	if len(split) != 3 {
		return nil, errors.New("Request line does not contain 3 parts")
	}
	method, err := parseMethod(split[0])
	if err != nil {
		return nil, err
	}
	uri, err := parseUri(split[1])
	if err != nil {
		return nil, err
	}
	major, minor, err := parseHttpVersion(split[2])
	if err != nil {
		return nil, err
	}

	req.Method = method

	req.Proto = "HTTP"
	req.ProtoMajor = major
	req.ProtoMinor = minor

	header := http.Header{}

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, io.EOF
		}
		if line == "\r\n" {
			break
		}
		line = strings.TrimSuffix(line, "\r\n")
		key, value, err := parseHeader(line)
		header.Add(key, value)
	}

	host := header.Get("Host")
	if host == "" {
		return nil, errors.New("Request does not contain Host header")
	}
	uri.Host = host
	req.URL = uri

	req.Header = header
	req.Body = io.NopCloser(reader)
	return &req, nil
}

func parseMethod(method string) (string, error) {
	switch method {
	case "GET":
		return "GET", nil
	case "POST":
		return "POST", nil
	case "PUT":
		return "PUT", nil
	case "DELETE":
		return "DELETE", nil
	case "PATCH":
		return "PATCH", nil
	case "HEAD":
		return "HEAD", nil
	case "OPTIONS":
		return "OPTIONS", nil
	case "CONNECT":
		return "CONNECT", nil
	case "TRACE":
		return "TRACE", nil
	default:
		return "", errors.New("Unknown method")
	}
}

func parseHttpVersion(version string) (major, minor int, err error) {
	rest, hasPrefix := strings.CutPrefix(version, "HTTP/")
	if !hasPrefix {
		return 0, 0, fmt.Errorf("Invalid HTTP Version: %s", version)
	}
	split := strings.Split(rest, ".")
	major, err = strconv.Atoi(split[0])
	if err != nil {
		return 0, 0, fmt.Errorf("Invalid major version: %s", version)
	}
	minor, err = strconv.Atoi(split[1])
	if err != nil {
		return 0, 0, fmt.Errorf("Invalid minor version: %s", version)
	}
	return major, minor, nil
}

func parseUri(uri string) (*url.URL, error) {
	return url.ParseRequestURI(uri)
}

func parseHeader(headerline string) (key string, value string, err error) {
	headers := strings.SplitN(headerline, ":", 2)
	if len(headers) != 2 {
		return "", "", fmt.Errorf("Invalid header line: %s", headerline)
	}
	return headers[0], headers[1], nil
}
