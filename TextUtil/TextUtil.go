package TextUtil

import (
	"flag"
	"github.com/golang/glog"
)

func init() {
	flag.Parse()
}

func Deflate(plainString string) {
	glog.V(3).Infof("input: " + plainString)

}
