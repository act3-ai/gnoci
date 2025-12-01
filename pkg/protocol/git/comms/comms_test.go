// Package comms facilitates receiving requests from and writing responses to Git via the remote helpers protocol.
//
// Protocol Reference: https://git-scm.com/docs/gitremote-helpers.
package comms

import (
	"bufio"
	"bytes"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/act3-ai/gnoci/pkg/protocol/git"
	"github.com/stretchr/testify/assert"
)

func TestNewCommunicator(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		in := strings.NewReader("foo")
		out := new(bytes.Buffer)

		comm := NewCommunicator(in, out)
		assert.NotNil(t, comm)
		defaultComm, ok := comm.(*defaultCommunicator)
		assert.True(t, ok)
		assert.NotNil(t, defaultComm)
		assert.NotNil(t, defaultComm.in)
		assert.Equal(t, out, defaultComm.out)
	})
}

func Test_defaultCommunicator_LookAhead(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)
	})
}

func Test_defaultCommunicator_previousOrNext(t *testing.T) {
	type fields struct {
		in       bufio.Scanner
		out      io.Writer
		previous []string
	}
	tests := []struct {
		name    string
		fields  fields
		want    []string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &defaultCommunicator{
				in:       tt.fields.in,
				out:      tt.fields.out,
				previous: tt.fields.previous,
			}
			got, err := c.previousOrNext()
			if (err != nil) != tt.wantErr {
				t.Errorf("defaultCommunicator.previousOrNext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("defaultCommunicator.previousOrNext() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_defaultCommunicator_next(t *testing.T) {
	type fields struct {
		in       bufio.Scanner
		out      io.Writer
		previous []string
	}
	tests := []struct {
		name    string
		fields  fields
		want    []string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &defaultCommunicator{
				in:       tt.fields.in,
				out:      tt.fields.out,
				previous: tt.fields.previous,
			}
			got, err := c.next()
			if (err != nil) != tt.wantErr {
				t.Errorf("defaultCommunicator.next() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("defaultCommunicator.next() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_defaultCommunicator_ParseCapabilitiesRequest(t *testing.T) {
	type fields struct {
		in       bufio.Scanner
		out      io.Writer
		previous []string
	}
	tests := []struct {
		name    string
		fields  fields
		want    *git.CapabilitiesRequest
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &defaultCommunicator{
				in:       tt.fields.in,
				out:      tt.fields.out,
				previous: tt.fields.previous,
			}
			got, err := c.ParseCapabilitiesRequest()
			if (err != nil) != tt.wantErr {
				t.Errorf("defaultCommunicator.ParseCapabilitiesRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("defaultCommunicator.ParseCapabilitiesRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_defaultCommunicator_ParseOptionRequest(t *testing.T) {
	type fields struct {
		in       bufio.Scanner
		out      io.Writer
		previous []string
	}
	tests := []struct {
		name    string
		fields  fields
		want    *git.OptionRequest
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &defaultCommunicator{
				in:       tt.fields.in,
				out:      tt.fields.out,
				previous: tt.fields.previous,
			}
			got, err := c.ParseOptionRequest()
			if (err != nil) != tt.wantErr {
				t.Errorf("defaultCommunicator.ParseOptionRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("defaultCommunicator.ParseOptionRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_defaultCommunicator_ParseListRequest(t *testing.T) {
	type fields struct {
		in       bufio.Scanner
		out      io.Writer
		previous []string
	}
	tests := []struct {
		name    string
		fields  fields
		want    *git.ListRequest
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &defaultCommunicator{
				in:       tt.fields.in,
				out:      tt.fields.out,
				previous: tt.fields.previous,
			}
			got, err := c.ParseListRequest()
			if (err != nil) != tt.wantErr {
				t.Errorf("defaultCommunicator.ParseListRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("defaultCommunicator.ParseListRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_defaultCommunicator_ParseFetchRequestBatch(t *testing.T) {
	type fields struct {
		in       bufio.Scanner
		out      io.Writer
		previous []string
	}
	tests := []struct {
		name    string
		fields  fields
		want    []git.FetchRequest
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &defaultCommunicator{
				in:       tt.fields.in,
				out:      tt.fields.out,
				previous: tt.fields.previous,
			}
			got, err := c.ParseFetchRequestBatch()
			if (err != nil) != tt.wantErr {
				t.Errorf("defaultCommunicator.ParseFetchRequestBatch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("defaultCommunicator.ParseFetchRequestBatch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_defaultCommunicator_ParsePushRequestBatch(t *testing.T) {
	type fields struct {
		in       bufio.Scanner
		out      io.Writer
		previous []string
	}
	tests := []struct {
		name    string
		fields  fields
		want    []git.PushRequest
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &defaultCommunicator{
				in:       tt.fields.in,
				out:      tt.fields.out,
				previous: tt.fields.previous,
			}
			got, err := c.ParsePushRequestBatch()
			if (err != nil) != tt.wantErr {
				t.Errorf("defaultCommunicator.ParsePushRequestBatch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("defaultCommunicator.ParsePushRequestBatch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_defaultCommunicator_WriteCapabilitiesResponse(t *testing.T) {
	type fields struct {
		in       bufio.Scanner
		out      io.Writer
		previous []string
	}
	type args struct {
		capabilities []git.Capability
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &defaultCommunicator{
				in:       tt.fields.in,
				out:      tt.fields.out,
				previous: tt.fields.previous,
			}
			if err := c.WriteCapabilitiesResponse(tt.args.capabilities); (err != nil) != tt.wantErr {
				t.Errorf("defaultCommunicator.WriteCapabilitiesResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_defaultCommunicator_WriteOptionResponse(t *testing.T) {
	type fields struct {
		in       bufio.Scanner
		out      io.Writer
		previous []string
	}
	type args struct {
		supported bool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &defaultCommunicator{
				in:       tt.fields.in,
				out:      tt.fields.out,
				previous: tt.fields.previous,
			}
			if err := c.WriteOptionResponse(tt.args.supported); (err != nil) != tt.wantErr {
				t.Errorf("defaultCommunicator.WriteOptionResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_defaultCommunicator_WriteListResponse(t *testing.T) {
	type fields struct {
		in       bufio.Scanner
		out      io.Writer
		previous []string
	}
	type args struct {
		resps []*git.ListResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &defaultCommunicator{
				in:       tt.fields.in,
				out:      tt.fields.out,
				previous: tt.fields.previous,
			}
			if err := c.WriteListResponse(tt.args.resps); (err != nil) != tt.wantErr {
				t.Errorf("defaultCommunicator.WriteListResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_defaultCommunicator_WritePushResponse(t *testing.T) {
	type fields struct {
		in       bufio.Scanner
		out      io.Writer
		previous []string
	}
	type args struct {
		resps []*git.PushResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &defaultCommunicator{
				in:       tt.fields.in,
				out:      tt.fields.out,
				previous: tt.fields.previous,
			}
			if err := c.WritePushResponse(tt.args.resps); (err != nil) != tt.wantErr {
				t.Errorf("defaultCommunicator.WritePushResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_defaultCommunicator_WriteFetchResponse(t *testing.T) {
	type fields struct {
		in       bufio.Scanner
		out      io.Writer
		previous []string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &defaultCommunicator{
				in:       tt.fields.in,
				out:      tt.fields.out,
				previous: tt.fields.previous,
			}
			if err := c.WriteFetchResponse(); (err != nil) != tt.wantErr {
				t.Errorf("defaultCommunicator.WriteFetchResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_defaultCommunicator_readLine(t *testing.T) {
	type fields struct {
		in       bufio.Scanner
		out      io.Writer
		previous []string
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &defaultCommunicator{
				in:       tt.fields.in,
				out:      tt.fields.out,
				previous: tt.fields.previous,
			}
			got, err := c.readLine()
			if (err != nil) != tt.wantErr {
				t.Errorf("defaultCommunicator.readLine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("defaultCommunicator.readLine() = %v, want %v", got, tt.want)
			}
		})
	}
}
