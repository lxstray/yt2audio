//TODO: поменять вызов yt-dlp и ffmpeg

package main

import (
	"encoding/json"
	"fmt"
	"image"
	"log"
	"net/http"
	"os"
	"os/exec"

	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func main() {

	e := echo.New()

	e.GET("/convert", Yt2mp3)

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "qq >-<, this is youtube converter")
	})

	e.Logger.Fatal(e.Start(":1323"))

}

func Yt2mp3(c echo.Context) error {
	url := c.QueryParam("url")
	if url == "" {
		return c.String(http.StatusBadRequest, "missing url parameter")
	}

	title, author, videoId := getInfo(url)

	audioPath, coverPath := generateTempFilesNames()

	// //cover
	coverUrl := fmt.Sprintf("https://img.youtube.com/vi/%s/maxresdefault.jpg", videoId)

	resp, err := http.Get(coverUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		coverUrl := fmt.Sprintf("https://img.youtube.com/vi/%s/hqdefault.jpg", videoId)

		resp, err := http.Get(coverUrl)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()

		img, err := imaging.Decode(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		croppedImg := imaging.Crop(img, image.Rect(105, 45, 105+270, 45+270))

		finalImg := imaging.Resize(croppedImg, 1600, 1600, imaging.Lanczos)

		err = imaging.Save(finalImg, coverPath)
		if err != nil {
			log.Fatal(err)
		}

	} else {
		img, err := imaging.Decode(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		croppedImg := imaging.CropCenter(img, min(img.Bounds().Dx(), img.Bounds().Dy()), min(img.Bounds().Dx(), img.Bounds().Dy()))

		finalImg := imaging.Resize(croppedImg, 1600, 1600, imaging.Lanczos)

		err = imaging.Save(finalImg, coverPath)
		if err != nil {
			log.Fatal(err)
		}
	}

	//audio
	audioCmd := exec.Command("./yt-dlp", "-x", "-f", "m4a", "--no-playlist", url, "-o", "-")
	audioPipe, err := audioCmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	//TODO: мб попробовать использовать fifo в линуксе что бы предать трек и картинку как поток

	ffmpegCmd := exec.Command("./ffmpeg", "-i", "pipe:0", "-i", coverPath, "-map", "0", "-map", "1", "-c", "copy", "-metadata", "artist="+author, "-metadata", "title="+title, "-disposition:v:0", "attached_pic", audioPath)
	ffmpegCmd.Stdin = audioPipe

	if err := audioCmd.Start(); err != nil {
		return err
	}

	if err := ffmpegCmd.Start(); err != nil {
		return err
	}

	if err := audioCmd.Wait(); err != nil {
		log.Fatal(err)
	}

	if err := ffmpegCmd.Wait(); err != nil {
		log.Fatal(err)
	}

	fileName := author + "_-_" + title + ".m4a"

	defer os.Remove(audioPath)
	defer os.Remove(coverPath)

	return c.Attachment(audioPath, fileName)
}

func generateTempFilesNames() (string, string) {
	tempAudio := "temp\\" + uuid.New().String() + ".m4a"
	tempCover := "temp\\" + uuid.New().String() + ".png"

	return tempAudio, tempCover
}

type VideoInfo struct {
	Title    string `json:"title"`
	Uploader string `json:"uploader"`
	VideoID  string `json:"id"`
}

func getInfo(url string) (string, string, string) {
	infoJSONCmd := exec.Command("./yt-dlp", "-j", url)

	infoJSON, err := infoJSONCmd.Output()
	if err != nil {
		fmt.Println("error infoJSONCmd:", err)
	}

	var info VideoInfo
	err = json.Unmarshal(infoJSON, &info)
	if err != nil {
		fmt.Println("error parsing JSON:", err)
	}

	return info.Title, info.Uploader, info.VideoID
}
