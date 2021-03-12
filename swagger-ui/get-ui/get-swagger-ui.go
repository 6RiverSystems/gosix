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
  "integrity": "sha512-hTNX6cX7KWtBZgk6ZQSOzsBJhqdCmD5NOIjb6dBPKSnYZidSkIXOcaPMR3+kwxLrj8bDC881bSDlNbLsHikacg==",
  "shasum": "4ed395b21f35a7b007c6e70ca3d5aa911e777f68",
  "tarball": "https://registry.npmjs.org/swagger-ui-dist/-/swagger-ui-dist-3.42.0.tgz",
  "fileCount": 20,
  "unpackedSize": 18500933,
  "npm-signature": "-----BEGIN PGP SIGNATURE-----\r\nVersion: OpenPGP.js v3.0.13\r\nComment: https://openpgpjs.org\r\n\r\nwsFcBAEBCAAQBQJgHDuhCRA9TVsSAnZWagAAEzIP/ig2VEO+uLp6bwrnBVsE\nMh1fiY8Jv2vDRlXPNtHFD6ZxFz8HxMj0rUpca3pTN6jqbvzyN9ktxf+UEBQg\nzqk89b3R5d+QtctlVm9pflS8BZyBPgRcPaS6KW3Hf4H0JcJgzpwwrfCHaJ2a\nDqj0cNLEJRfwnn39zFQh+i+cTzwLj5OlAOsNc761lsVtqUJ9a1oIsyBJHF90\nkFC+svRZoAd0hb+0zGYV9ujdpHmagsgzjDt3PcCFdA9DIAIle1xawK4MUMn0\nh0mjlEG8P4kF365KLbBS0qlSjxBcJS1Tv0wYCLI1EAzXIxw+Dc4cz6004kJq\ngmc+8ue9T2KYNMRZaFXDGrJJ/CNw8vxut9laql4Ub2brO+VMY7CwBEqc1goL\nPOB1sYig148NPEkR9yXh9CBWRMWn596ZAFOfgAqieQXQa0dn7RhH1S/Gs8TS\nSydOZa5sOyNl9IXAggudYPWx110VQqWVHCCdeGjvz8Zgm12njWJ+9aTBAbl8\nKOFky7TjC+Jm0xGg/AbGPS7Cb2MB5M2zpki+JKmqi1zp0v+hcWAYndgykTpv\n/mc2guzR78DAb9pih1HW5mgk51BwOjQoDRQ44IscABIVqj32wx1sFR7slcdP\nQObZXNDxlQKs4FNsy7a9KtdkuzjuBETEB8SzeJ87lU3o9CPY24f7GP2WPnvV\ni/wZ\r\n=IBgz\r\n-----END PGP SIGNATURE-----\r\n"
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
