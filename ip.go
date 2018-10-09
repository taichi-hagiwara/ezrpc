package ezrpc

import (
	"fmt"
	"net"
	"strconv"

	"github.com/pkg/errors"
)

// IPEndPoint は、ホストとポート番号を結合する。
type IPEndPoint struct {
	Host string
	Port uint16
}

func (e IPEndPoint) String() string {
	return fmt.Sprintf("%s:%d", e.Host, e.Port)
}

// ParseIPEndPoint は、文字列から ParseIPEndPoint を取得する。
func ParseIPEndPoint(str string) (IPEndPoint, error) {
	host, port, err := net.SplitHostPort(str)
	if err != nil {
		return IPEndPoint{}, errors.Wrapf(err, "failed to parse IPEndPoint: %s", str)
	}
	iport, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return IPEndPoint{}, errors.Wrapf(err, "failed to parse port: %s", port)
	}
	return IPEndPoint{Host: host, Port: uint16(iport)}, nil

}
