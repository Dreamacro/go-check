package executor

import (
	"bytes"

	"github.com/francoispqt/gojay"
)

type Update struct {
	Path    string
	Version string
}

func (u *Update) NKeys() int {
	return 2
}

func (u *Update) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case "Path":
		return dec.AddString(&u.Path)
	case "Version":
		return dec.AddString(&u.Version)
	}
	return nil
}

type Package struct {
	Path     string
	Main     bool
	Indirect bool
	Version  string
	Update   *Update
}

func (e *Package) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case "Path":
		return dec.AddString(&e.Path)
	case "Main":
		return dec.AddBool(&e.Main)
	case "Indirect":
		return dec.AddBool(&e.Indirect)
	case "Version":
		return dec.AddString(&e.Version)
	case "Update":
		e.Update = &Update{}
		return dec.AddObject(e.Update)
	}
	return nil
}

func (e *Package) NKeys() int {
	return 5
}

type channelStream chan *Package

func (c channelStream) UnmarshalStream(dec *gojay.StreamDecoder) error {
	elm := &Package{}
	if err := dec.Object(elm); err != nil {
		return err
	}
	c <- elm
	return nil
}

func Scan(output []byte) []*Package {
	streamChan := channelStream(make(chan *Package))
	dec := gojay.Stream.BorrowDecoder(bytes.NewBuffer(output))
	go dec.DecodeStream(streamChan)
	packages := []*Package{}
	for {
		select {
		case v := <-streamChan:
			packages = append(packages, v)
		case <-dec.Done():
			return packages
		}
	}
}
