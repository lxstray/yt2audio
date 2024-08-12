//TODO: поменять вызов yt-dlp и ffmpeg

package main

import (
	"encoding/json"
	"fmt"
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

	tempAudioPath, audioPath, coverPath := generateTempFilesNames()

	//TODO: если status code 200 для maxres то делать hq

	// //cover
	coverUrl := fmt.Sprintf("https://img.youtube.com/vi/%s/maxresdefault.jpg", videoId)

	resp, err := http.Get(coverUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

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

	//audio
	audioCmd := exec.Command("./yt-dlp", "-x", "-f", "m4a", "--no-playlist", url, "-o", tempAudioPath)

	if err := audioCmd.Start(); err != nil {
		return err
	}

	if err := audioCmd.Wait(); err != nil {
		log.Fatal(err)
	}

	ffmpegCmd := exec.Command("./ffmpeg", "-i", tempAudioPath, "-i", coverPath, "-map", "0", "-map", "1", "-c", "copy", "-metadata", "artist="+author, "-metadata", "title="+title, "-disposition:v:0", "attached_pic", audioPath)

	if err := ffmpegCmd.Start(); err != nil {
		return err
	}

	if err := ffmpegCmd.Wait(); err != nil {
		log.Fatal(err)
	}

	fileName := author + "_-_" + title + ".m4a"

	defer os.Remove(audioPath)
	defer os.Remove(coverPath)
	defer os.Remove(tempAudioPath)

	return c.Attachment(audioPath, fileName)
}

func generateTempFilesNames() (string, string, string) {
	tempAudioTemp := "temp\\" + uuid.New().String() + "-temp" + ".m4a"
	tempAudio := "temp\\" + uuid.New().String() + ".m4a"
	tempCover := "temp\\" + uuid.New().String() + ".png"

	return tempAudioTemp, tempAudio, tempCover
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
