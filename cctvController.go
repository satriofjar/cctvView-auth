package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/deepch/vdk/av"

	webrtc "github.com/deepch/vdk/format/webrtcv3"
	"github.com/gin-gonic/gin"
)

type JCodec struct {
	Type string
}

// HTTPAPIServerIndex  index
func HTTPAPIServerIndex(c *gin.Context) {
	_, all := Config.list()
	sort.Strings(all)
	if len(all) > 0 {
		c.Header("Cache-Control", "no-cache, max-age=0, must-revalidate, no-store")
		// c.Redirect(http.StatusMovedPermanently, "stream/player/"+all[0])
		c.Redirect(http.StatusMovedPermanently, "/stream/floor/lantai_1")
	} else {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"port":    Config.Server.HTTPPort,
			"version": time.Now().String(),
		})
	}
}

// handle login view
func HTTPAPIServerLogin(c *gin.Context) {
	c.HTML(http.StatusOK, "login.tmpl", gin.H{
		"port": Config.Server.HTTPPort,
	})
}

func HTTPAPIServerFloor(c *gin.Context) {
	streams := Config.ListStreamsByFloor(c.Param("uuid"))
	sort.Strings(streams)
	floor := strings.ReplaceAll(c.Param("uuid"), "_", " ")
	c.HTML(http.StatusOK, "main.html", gin.H{
		// "port":     Config.Server.HTTPPort,
		"suuidMap": streams,
		"floor":    floor,
		// "version":  time.Now().String(),
	})
}

// HTTPAPIServerStreamPlayer stream player
func HTTPAPIServerStreamPlayer(c *gin.Context) {
	_, all := Config.list()
	sort.Strings(all)
	f := Config.getFloor(c.Param("uuid"))
	floor := strings.ReplaceAll(f, "_", " ")
	// streams := Config.ListStreamsByFloor(some)
	// sort.Strings(streams)
	c.HTML(http.StatusOK, "player.html", gin.H{
		"port":  Config.Server.HTTPPort,
		"suuid": c.Param("uuid"),
		"floor": floor,
		// "suuidMap": all,
		// "suuidMap": streams,
		"version": time.Now().String(),
	})
}

func HTTPAPIServerThumbnail(c *gin.Context) {
	url := Config.getUrl(c.Param("uuid"))
	if url == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL is required"})
		return
	}

	// Membuat channel untuk mengirim hasil thumbnail dan error
	thumbnailChan := make(chan []byte)
	errorChan := make(chan error)

	// Menjalankan fungsi generateThumbnail secara konkurensi
	go generateThumbnail(url, thumbnailChan, errorChan)

	// Menggunakan select untuk menunggu hasil thumbnail atau error
	select {
	case thumbnail := <-thumbnailChan:
		c.Data(http.StatusOK, "image/jpeg", thumbnail)
	case err := <-errorChan:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

func generateThumbnail(url string, thumbnailChan chan []byte, errorChan chan error) {
	// cmd := exec.Command("ffmpeg", "-i", url, "-vf", "thumbnail,scale=320:240", "-frames:v", "1", "-f", "image2", "-")
	cmd := exec.Command("ffmpeg", "-i", url, "-vframes", "1", "-ss", "00:00:01", "-s", "320x240", "-f", "image2", "-")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		errorChan <- fmt.Errorf("failed to generate thumbnail: %v", err)
		return
	}

	// Mengirim hasil thumbnail ke channel
	thumbnailChan <- out.Bytes()
}

// HTTPAPIServerStreamCodec stream codec
func HTTPAPIServerStreamCodec(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
	if Config.ext(c.Param("uuid")) {
		Config.RunIFNotRun(c.Param("uuid"))
		codecs := Config.coGe(c.Param("uuid"))
		if codecs == nil {
			return
		}
		var tmpCodec []JCodec
		for _, codec := range codecs {
			if codec.Type() != av.H264 && codec.Type() != av.PCM_ALAW && codec.Type() != av.PCM_MULAW && codec.Type() != av.OPUS {
				log.Println("Codec Not Supported WebRTC ignore this track", codec.Type())
				continue
			}
			if codec.Type().IsVideo() {
				tmpCodec = append(tmpCodec, JCodec{Type: "video"})
			} else {
				tmpCodec = append(tmpCodec, JCodec{Type: "audio"})
			}
		}
		b, err := json.Marshal(tmpCodec)
		if err == nil {
			_, err = c.Writer.Write(b)
			if err != nil {
				log.Println("Write Codec Info error", err)
				return
			}
		}
	}
}

// HTTPAPIServerStreamWebRTC stream video over WebRTC
func HTTPAPIServerStreamWebRTC(c *gin.Context) {
	if !Config.ext(c.PostForm("suuid")) {
		log.Println("Stream Not Found")
		return
	}
	Config.RunIFNotRun(c.PostForm("suuid"))
	codecs := Config.coGe(c.PostForm("suuid"))
	if codecs == nil {
		log.Println("Stream Codec Not Found")
		return
	}
	var AudioOnly bool
	if len(codecs) == 1 && codecs[0].Type().IsAudio() {
		AudioOnly = true
	}
	muxerWebRTC := webrtc.NewMuxer(webrtc.Options{ICEServers: Config.GetICEServers(), PortMin: Config.GetWebRTCPortMin(), PortMax: Config.GetWebRTCPortMax()})
	answer, err := muxerWebRTC.WriteHeader(codecs, c.PostForm("data"))
	if err != nil {
		log.Println("WriteHeader", err)
		return
	}
	_, err = c.Writer.Write([]byte(answer))
	if err != nil {
		log.Println("Write", err)
		return
	}
	go func() {
		cid, ch := Config.clAd(c.PostForm("suuid"))
		defer Config.clDe(c.PostForm("suuid"), cid)
		defer muxerWebRTC.Close()
		var videoStart bool
		noVideo := time.NewTimer(20 * time.Second)
		for {
			select {
			case <-noVideo.C:
				log.Println("noVideo")
				return
			case pck := <-ch:
				if pck.IsKeyFrame || AudioOnly {
					noVideo.Reset(20 * time.Second)
					videoStart = true
				}
				if !videoStart && !AudioOnly {
					continue
				}
				err = muxerWebRTC.WritePacket(pck)
				if err != nil {
					log.Println("WritePacket", err)
					return
				}
			}
		}
	}()
}
