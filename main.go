package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gokyle/filecache"
	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v3"
	"go.uber.org/multierr"
)

const section_dir = "sections"

var cache *filecache.FileCache

type manifestFile struct {
	master   []byte
	variants map[string][]byte
}

type getFragRequest struct {
	Name string `json:"name"`
}

const chunkSize = 64 * 1024 // 64KB max

func getManifestFile() (*manifestFile, error) {
	m := manifestFile{}
	m.variants = make(map[string][]byte)
	var err error
	m.master, err = os.ReadFile("sections/master.m3u8")
	if err != nil {
		return nil, err
	}
	// get variant filenames
	re := regexp.MustCompile("(?m)^.*?m3u8$")
	matches := re.FindAllString(string(m.master), -1)
	for _, match := range matches {
		variantContents, err := os.ReadFile("sections/" + strings.Replace(match, "pref:", "", -1))
		if err != nil {
			return nil, err
		}
		m.variants[match] = variantContents
	}
	return &m, nil
}

// todo: retry sends :/
func sendChunkedWrtc(d *webrtc.DataChannel, typ string, msg io.Reader) error {
	var err error
	// send start
	d.SendText(typ)
	// start streaming
	buf := make([]byte, chunkSize)
	for {
		n, readErr := msg.Read(buf)
		if err == io.EOF || n == 0 {
			break
		}
		if err != nil {
			err = multierr.Append(err, readErr)
			goto er
		}
		if sendErr := d.Send(buf[:n]); err != nil {
			err = multierr.Append(err, sendErr)
			goto er
		}
	}
	// send end
	if sendErr := d.SendText("END"); err != nil {
		err = multierr.Append(err, sendErr)
		println("datachannel, failed to send end. err: ", err.Error())
	}
	return err
er:
	// send err
	if sendErr := d.SendText("ERR"); err != nil {
		err = multierr.Append(err, sendErr)
		println("datachannel, failed to send error. err: ", err.Error())
	}
	return err
}

func Serve(api *webrtc.API, manifest *manifestFile) {
	driver := gin.Default()
	driver.LoadHTMLGlob("templates/*")
	driver.Static("/static", "./statics")
	driver.Static("/keys", "./keys")
	driver.GET("/", func(ctx *gin.Context) {
		ctx.HTML(http.StatusOK, "main.tmpl", gin.H{})
	})
	driver.POST("signal", func(ctx *gin.Context) {
		println("got signaling request")
		peerConnection, err := api.NewPeerConnection(webrtc.Configuration{})
		if err != nil {
			println("could not create peer connection: ", err.Error())
			ctx.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		// Set the handler for ICE connection state
		peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
			println("ICE Connection State has changed: ", connectionState.String())
		})
		peerConnection.OnConnectionStateChange(func(pcs webrtc.PeerConnectionState) {
			println("Connection State has changed: ", pcs.String())
		})
		peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
			d.OnOpen(func() {
				if err := sendChunkedWrtc(d, "MASTER", bytes.NewReader(manifest.master)); err != nil {
					println("could not send master manifest: ", err.Error())
				}
				for variantName, variantContent := range manifest.variants {
					if err := sendChunkedWrtc(d, "VARIANT_"+variantName, bytes.NewReader(variantContent)); err != nil {
						println("could not send variant "+variantName+": ", err.Error())
					}
				}
				d.SendText("CLOSING")
			})
			d.OnBufferedAmountLow(func() {
				go peerConnection.Close()
			})
		})
		var offer webrtc.SessionDescription
		if err = json.NewDecoder(ctx.Request.Body).Decode(&offer); err != nil {
			println("could not decode offer: ", err.Error())
			ctx.AbortWithStatus(http.StatusBadRequest)
			return
		}

		if err = peerConnection.SetRemoteDescription(offer); err != nil {
			println("could not set remote description: ", err.Error())
			ctx.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		// Create channel that is blocked until ICE Gathering is complete
		gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

		answer, err := peerConnection.CreateAnswer(nil)
		if err != nil {
			println("could not create answer: ", err.Error())
			ctx.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		if err = peerConnection.SetLocalDescription(answer); err != nil {
			println("could not set local description: ", err.Error())
			ctx.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		<-gatherComplete
		// println(*&peerConnection.LocalDescription().SDP)
		ctx.JSON(http.StatusOK, *peerConnection.LocalDescription())
	})
	driver.POST("/frag", func(ctx *gin.Context) {
		var req getFragRequest
		ctx.BindJSON(&req)
		println(req.Name)
		cont, err := cache.ReadFile(path.Join(section_dir, req.Name))
		if err != nil {
			ctx.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		ctx.Data(http.StatusOK, "application/octet-stream", cont)
	})
	driver.Run("0.0.0.0:8080")
}

func main() {
	manifest, err := getManifestFile()
	if err != nil {
		panic(err)
	}
	cache = filecache.NewDefaultCache()

	mediaEngine := webrtc.MediaEngine{}
	settingsEngine := webrtc.SettingEngine{}
	settingsEngine.SetLite(true)
	interceptorRegistry := &interceptor.Registry{}
	api := webrtc.NewAPI(
		webrtc.WithMediaEngine(&mediaEngine),
		webrtc.WithInterceptorRegistry(interceptorRegistry),
		webrtc.WithSettingEngine(settingsEngine),
	)
	Serve(api, manifest)
}
