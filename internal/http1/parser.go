package http1

import (
	"bufio"
	"errors"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
)

var (
	Incomplete = errors.New("incomplete stream")
)

func ParseRequest(reader *bufio.Reader) (*http.Request, error) {

	req := http.Request{}

	requestLine, err := reader.ReadString('\n')
	if err != nil {
		if len(requestLine) == 0 {
			return nil, io.EOF
		}
		return nil, ClientError{BadRequest}
	}
	requestLine, found := strings.CutSuffix(requestLine, "\r\n")
	if !found {
		return nil, ClientError{BadRequest}
	}

	split := strings.Split(requestLine, " ")
	if len(split) != 3 {
		return nil, ClientError{BadRequest}
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
	numHeaders := 0
	for {

		if numHeaders > 100 {
			return nil, ClientError{HeaderTooLarge}
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, ClientError{BadRequest}
		}
		if len(line) > 16*1024 {
			return nil, ClientError{HeaderTooLarge}
		}
		if line == "\r\n" {
			break
		}
		line = strings.TrimSuffix(line, "\r\n")
		key, value, err := parseHeader(line)
		if err != nil {
			return nil, err
		}
		header.Add(key, value)
		numHeaders++
	}

	host := header.Get("Host")
	if host != "" {
		uri.Host = host
	}
	var bodyReader io.Reader = io.LimitReader(reader, 0)
	req.ContentLength = 0
	trailers := false

	contentLengths := header.Values("Content-Length")

	for _, contentLength := range contentLengths {
		if contentLength != "" {
			length, err := strconv.ParseInt(contentLength, 10, 64)
			if err != nil {
				return nil, ClientError{BadRequest}
			}
			if req.ContentLength == 0 {
				req.ContentLength = length
			} else {
				if length != req.ContentLength {
					return nil, ClientError{BadRequest}
				}
			}
			bodyReader = io.LimitReader(reader, req.ContentLength)

		}
	}

	if transferEncoding := header.Get("Transfer-Encoding"); strings.ToLower(transferEncoding) == "chunked" {
		bodyReader = httputil.NewChunkedReader(reader)
		req.ContentLength = -1
		trailers = true

	}

	req.URL = uri

	req.Header = header
	req.Body = ClosingReader(bodyReader, reader, trailers)
	return &req, nil
}

func parseMethod(method string) (string, error) {
	switch strings.ToUpper(method) {
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
		return "", ClientError{BadRequest}
	}
}

func parseHttpVersion(version string) (major, minor int, err error) {
	rest, hasPrefix := strings.CutPrefix(version, "HTTP/")
	if !hasPrefix {
		return 0, 0, ClientError{BadRequest}
	}
	majorS, minorS, found := strings.Cut(rest, ".")
	if !found {
		return 0, 0, ClientError{BadRequest}
	}
	major, err = strconv.Atoi(majorS)
	if err != nil {
		return 0, 0, ClientError{BadRequest}
	}
	minor, err = strconv.Atoi(minorS)
	if err != nil {
		return 0, 0, ClientError{BadRequest}
	}

	if major != 1 || (minor != 1 && minor != 0) {
		return 0, 0, ServerError{NotSupported}
	}

	return major, minor, nil
}

func parseUri(uri string) (*url.URL, error) {
	return url.ParseRequestURI(uri)
}

func parseHeader(headerline string) (key string, value string, err error) {
	key, value, found := strings.Cut(headerline, ":")
	if !found {
		return "", "", ClientError{BadRequest}
	}

	if !isValidHeaderName(key) || !isValidHeaderValue(value) {
		return "", "", ClientError{BadRequest}
	}

	return key, strings.TrimSpace(value), nil
}

func isValidHeaderName(key string) bool {
	if key == "" {
		return false
	}

	return true
}

func isValidHeaderValue(value string) bool {

	return true
}

type closingReader struct {
	bodyReader io.Reader
	connReader *bufio.Reader
	sawEOF     bool
	trailers   bool
}

func (cr *closingReader) Read(p []byte) (n int, err error) {
	n, err = cr.bodyReader.Read(p)
	if err == io.EOF {
		cr.sawEOF = true
	}
	return n, err
}

func (cr *closingReader) Close() error {
	if !cr.sawEOF {
		_, err := io.Copy(io.Discard, cr.bodyReader)
		if err != nil {
			return Incomplete
		}
	}
	cr.sawEOF = true
	if cr.trailers {

		for {
			line, err := cr.connReader.ReadString('\n')
			if err != nil {
				return Incomplete
			}
			if line == "\r\n" {
				break
			}
			line = strings.TrimSuffix(line, "\r\n")

		}

	}

	return nil
}

func ClosingReader(r io.Reader, cr *bufio.Reader, trailers bool) io.ReadCloser {
	return &closingReader{r, cr, false, trailers}
}
