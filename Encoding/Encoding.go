package Encoding

import (
	"bytes"
	"io/ioutil"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

/*
HtmlContentEncoding 是一个接口
*/
type HtmlContentEncoding interface {
	ConvertToUtf8(s []byte) ([]byte, error)
}

/*
GbkToUtf8 从 gbk 转码为 utf8
*/
func GbkToUtf8(s []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(s), simplifiedchinese.GBK.NewDecoder())
	d, e := ioutil.ReadAll(reader)
	if e != nil {
		return nil, e
	}
	return d, nil
}
