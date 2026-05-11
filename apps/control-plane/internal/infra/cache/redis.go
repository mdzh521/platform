package cache

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type RedisStore struct {
	client *redisConn
}

func NewRedisStore(rawURL string) (*RedisStore, error) {
	client, err := newRedisConn(rawURL)
	if err != nil {
		return nil, err
	}
	return &RedisStore{client: client}, nil
}

func (s *RedisStore) Driver() string { return "redis" }

func (s *RedisStore) Ping(ctx context.Context) error {
	_, err := s.client.command(ctx, "PING")
	return err
}

func (s *RedisStore) Get(ctx context.Context, key string) (string, bool, error) {
	reply, err := s.client.command(ctx, "GET", key)
	if err != nil {
		if errors.Is(err, errRedisNil) {
			return "", false, nil
		}
		return "", false, err
	}
	return reply, true, nil
}

func (s *RedisStore) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if ttl > 0 {
		_, err := s.client.command(ctx, "SET", key, value, "PX", strconv.FormatInt(ttl.Milliseconds(), 10))
		return err
	}
	_, err := s.client.command(ctx, "SET", key, value)
	return err
}

func (s *RedisStore) Delete(ctx context.Context, key string) error {
	_, err := s.client.command(ctx, "DEL", key)
	return err
}

func (s *RedisStore) Push(ctx context.Context, list, value string) error {
	_, err := s.client.command(ctx, "LPUSH", list, value)
	return err
}

func (s *RedisStore) Close() error { return nil }

var errRedisNil = errors.New("redis nil")

type redisConn struct {
	address  string
	password string
	db       int
	timeout  time.Duration
}

func newRedisConn(rawURL string) (*redisConn, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if parsed.Scheme != "redis" && parsed.Scheme != "rediss" {
		return nil, fmt.Errorf("unsupported redis scheme: %s", parsed.Scheme)
	}
	db := 0
	if parsed.Path != "" && parsed.Path != "/" {
		value := strings.TrimPrefix(parsed.Path, "/")
		db, err = strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("invalid redis db: %s", value)
		}
	}
	password, _ := parsed.User.Password()
	address := parsed.Host
	if !strings.Contains(address, ":") {
		address += ":6379"
	}
	return &redisConn{
		address:  address,
		password: password,
		db:       db,
		timeout:  3 * time.Second,
	}, nil
}

func (c *redisConn) command(ctx context.Context, args ...string) (string, error) {
	conn, err := (&net.Dialer{Timeout: c.timeout}).DialContext(ctx, "tcp", c.address)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(c.timeout))
	if c.password != "" {
		if err := writeRESP(conn, "AUTH", c.password); err != nil {
			return "", err
		}
		if _, err := readRESP(conn); err != nil {
			return "", err
		}
	}
	if c.db > 0 {
		if err := writeRESP(conn, "SELECT", strconv.Itoa(c.db)); err != nil {
			return "", err
		}
		if _, err := readRESP(conn); err != nil {
			return "", err
		}
	}
	if err := writeRESP(conn, args...); err != nil {
		return "", err
	}
	return readRESP(conn)
}

func writeRESP(conn net.Conn, args ...string) error {
	var builder strings.Builder
	builder.WriteString("*")
	builder.WriteString(strconv.Itoa(len(args)))
	builder.WriteString("\r\n")
	for _, arg := range args {
		builder.WriteString("$")
		builder.WriteString(strconv.Itoa(len(arg)))
		builder.WriteString("\r\n")
		builder.WriteString(arg)
		builder.WriteString("\r\n")
	}
	_, err := conn.Write([]byte(builder.String()))
	return err
}

func readRESP(conn net.Conn) (string, error) {
	buf := make([]byte, 1)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return "", err
	}
	line, err := readRESPLine(conn)
	if err != nil {
		return "", err
	}
	switch buf[0] {
	case '+', ':':
		return line, nil
	case '$':
		size, err := strconv.Atoi(line)
		if err != nil {
			return "", err
		}
		if size < 0 {
			return "", errRedisNil
		}
		payload := make([]byte, size+2)
		if _, err := io.ReadFull(conn, payload); err != nil {
			return "", err
		}
		return string(payload[:size]), nil
	case '-':
		return "", errors.New(line)
	default:
		return "", fmt.Errorf("unsupported redis reply: %q", buf[0])
	}
}

func readRESPLine(conn net.Conn) (string, error) {
	var out strings.Builder
	buf := make([]byte, 1)
	for {
		if _, err := io.ReadFull(conn, buf); err != nil {
			return "", err
		}
		if buf[0] == '\r' {
			if _, err := io.ReadFull(conn, buf); err != nil {
				return "", err
			}
			return out.String(), nil
		}
		out.WriteByte(buf[0])
	}
}
