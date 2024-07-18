// Copyright 2017-2023 Fortio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build !no_http && !no_net && tinygo
// +build !no_http,!no_net,tinygo

package log

import (
	"crypto/tls"
	"fmt"
	"net/http"
)

// TLSInfo on tiny go doesn't support printing peer certificate info.
// Use [AppendTLSInfoAttrs] unless you do want to just output text.
func TLSInfo(r *http.Request) string {
	if r.TLS == nil {
		return ""
	}
	return fmt.Sprintf(" https %s", tls.CipherSuiteName(r.TLS.CipherSuite))
}

func AppendTLSInfoAttrs(attrs []KeyVal, r *http.Request) []KeyVal {
	if r.TLS == nil {
		return attrs
	}
	attrs = append(attrs, Attr("tls", true))
	return attrs
}
