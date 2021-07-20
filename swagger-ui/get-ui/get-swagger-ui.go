// Copyright (c) 2021 6 River Systems
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

// this isn't part of the actual code build
// +build generate

// it needs to be in a separate directory to keep vscode and gopls from getting
// angry about package name mismatches

package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path"
	"regexp"

	"github.com/pkg/errors"
)

// update this by running: `npm view swagger-ui-dist@latest --json .dist`
var swaggerUiDistInfo = `{
  "integrity": "sha512-ueaZ45OHhHvGKmocvCkxFY8VCfbP5PgcxutoQxy9j8/VZeDoLDvg8FBf4SO6NxHhieNAdYPUd0O6G9FjJO2fqw==",
  "shasum": "f08d2c9b4a2dce922ba363c598e4795b5ccf0b80",
  "tarball": "https://registry.npmjs.org/swagger-ui-dist/-/swagger-ui-dist-3.46.0.tgz",
  "fileCount": 20,
  "unpackedSize": 18917597,
  "npm-signature": "-----BEGIN PGP SIGNATURE-----\r\nVersion: OpenPGP.js v3.0.13\r\nComment: https://openpgpjs.org\r\n\r\nwsFcBAEBCAAQBQJgZMUFCRA9TVsSAnZWagAAZKcP/14/rrHJgUel//wjq6zk\nKZEXL3lMJhxleZP/Zum5MJmVOaEJHhttbkYdo/o+YL7E5CUvDg3dzk3ryTtB\nX+oG1Pp5rGDLBoc62l+V3Y7iEAM5j86kf1lTmPBt7ua2sxDb0WIFNKa3GUFw\n7BZuWCGx78lifw5xJAKjp4sNe72twn+y4ZyUPhz5OL/owxyFlAs+5zZzia9x\nHSTt3zhUUFIG8EPYA29x+wH97KIBh99zGgtvvLalk1NbH6lmr7HxHmp6EHnX\nPjV54KUalJUgUBdBYIAdAEP9v+5oQg8CxqDac5sAYMI3r13xgSEDcYk8GQZt\n1FG30f/APBhHDUuMD7peZV9graXYTHKh5FCQq4cnRy4GsbhS9UpJFVFXWAS1\n/0pH1kkcEuQwTwIuCq1jnQKWxJA77ETJtqqfjoYxiUBevN7m+cfhiPLGrGpz\njO6VXnjKaInq0SMa8Y19DPpMK5xWnVmwB+6Xiw+x8PKlpiBoaFRmnQ/NIrqB\npoV+naVODuDeRuCdnUr4HYp1Y9Qv58HmukPRqf+KyHvyWhHL/az1ZTAmLvsF\nduxTe7FsOzc4S77WF9QwFlsbiHrNl99xxoEPN9/GQy1cw9kw0bfIHtQ5wsVL\nrTvncyfGmnt7d5AvO2ebq9HprkzRRRPrjTlQEBDNr9ZtxaDtEd4kueLspCoN\nUDWi\r\n=raRa\r\n-----END PGP SIGNATURE-----\r\n"
}`

func main() {
	var distInfo map[string]interface{}
	if err := json.Unmarshal([]byte(swaggerUiDistInfo), &distInfo); err != nil {
		panic(err)
	}
	resp, err := http.Get(distInfo["tarball"].(string))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		panic("Failed to get swagger-ui-dist tarball")
	}
	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	actualSha1 := sha1.Sum(buf)
	correctSha1, err := hex.DecodeString(distInfo["shasum"].(string))
	if err != nil {
		panic(err)
	}
	if !bytes.Equal(actualSha1[:], correctSha1) {
		panic(errors.Errorf("swagger-ui-dist sha1 mismatch, want %s got %s", distInfo["shasum"], hex.EncodeToString(actualSha1[:])))
	}
	zReader, err := gzip.NewReader(bytes.NewReader(buf))
	if err != nil {
		panic(err)
	}
	err = os.Mkdir("ui", 0777) // rely on umask
	if err != nil && !errors.Is(err, os.ErrExist) {
		panic(err)
	}
	tReader := tar.NewReader(zReader)
	for {
		h, err := tReader.Next()
		if h == nil || err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}

		if h.Typeflag != tar.TypeReg {
			continue
		}

		// we only want certain files
		switch h.Name {
		case "package/index.html":
			// we "patch" this one in-place
			fileData, err := io.ReadAll(tReader)
			if err != nil {
				panic(err)
			}
			fileContent := string(fileData)
			// replace the "call region" with a custom invocation that loads config
			// from a URL, so that apps can customize it
			callRegionMatch := regexp.MustCompile(`(?s)// Begin Swagger UI call region.*// End Swagger UI call region`)
			fileContent = callRegionMatch.ReplaceAllString(
				fileContent,
				// TODO: share the configUrl path with the main swaggerui package
				// the presets and plugin bits are code, not json, so we can't move them
				// into the configUrl. other bits stay here because of
				// https://github.com/swagger-api/swagger-ui/issues/4455 causing them to
				// not load correctly from the url
				`const ui = SwaggerUIBundle({
					configUrl: '../oas-ui-config',
					dom_id: '#swagger-ui',
					deepLinking: true,
					layout: 'StandaloneLayout',
					presets: [
						SwaggerUIBundle.presets.apis,
						SwaggerUIStandalonePreset
					],
					plugins: [
						SwaggerUIBundle.plugins.DownloadUrl
					],
				})`,
			)
			if err = os.WriteFile(path.Join("ui", path.Base(h.Name)), []byte(fileContent), 0666); err != nil { // umask again
				panic(err)
			}
		case "package/favicon-16x16.png",
			"package/favicon-32x32.png",
			"package/swagger-ui-bundle.js",
			// "package/swagger-ui-bundle.js.map",
			"package/swagger-ui-standalone-preset.js",
			// "package/swagger-ui-standalone-preset.js.map",
			"package/swagger-ui.css":
			// write this into the ui dir
			fileData, err := io.ReadAll(tReader)
			if err != nil {
				panic(err)
			}
			if err = os.WriteFile(path.Join("ui", path.Base(h.Name)), fileData, 0666); err != nil { // umask again
				panic(err)
			}

		}
	}
}
