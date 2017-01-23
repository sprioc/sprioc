package core

import (
	"bytes"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/sprioc/composer/pkg/contentStorage"
	"github.com/sprioc/composer/pkg/metadata"
	"github.com/sprioc/composer/pkg/model"
	"github.com/sprioc/composer/pkg/redis"
	"github.com/sprioc/composer/pkg/refs"
	"github.com/sprioc/composer/pkg/rsp"
)

// TODO NEED TO ABSTRACT THIS FURTHER

func UploadImage(user model.User, file []byte) rsp.Response {

	var sc model.Ref
	var err error
	if sc, err = redis.GenerateShortCode(model.Images); err != nil {
		return rsp.Response{Code: http.StatusInternalServerError}
	}

	img := model.Image{
		ShortCode:    sc,
		Owner:        user.ShortCode,
		PublishTime:  time.Now().Unix(),
		LastModified: time.Now().Unix(),
	}

	n := len(file)

	if n == 0 {
		return rsp.Response{Message: "Cannot upload file with 0 bytes.", Code: http.StatusBadRequest}
	}

	err = contentStorage.ProccessImage(file, n, img.ShortCode.ShortCode, "content")
	if err != nil {
		log.Println(err)
		return rsp.Response{Message: err.Error(), Code: http.StatusBadRequest}
	}

	buf := bytes.NewBuffer(file)

	metadata.GetMetadata(buf, &img)

	err = redis.CreateImage(user.GetRef(), img)
	if err != nil {
		return rsp.Response{Message: "Error while adding image to DB", Code: http.StatusInternalServerError}
	}

	// go metadata.SetLocation(img.MetaData.Location)

	return rsp.Response{Code: http.StatusAccepted, Data: map[string]string{"link": refs.GetURL(img.ShortCode)}}
}

func UploadAvatar(user model.User, file []byte) rsp.Response {
	n := len(file)

	if n == 0 {
		return rsp.Response{Message: "Cannot upload file with 0 bytes.", Code: http.StatusBadRequest}
	}

	// REVIEW check that you can overwrite on aws
	err := contentStorage.ProccessImage(file, n, user.ShortCode.ShortCode, "avatar")
	if err != nil {
		log.Println(err)
		return rsp.Response{Message: err.Error(), Code: http.StatusBadRequest}
	}

	return rsp.Response{Code: http.StatusAccepted}
}

func formatSources(shortcode, location string) model.ImgSource {
	var prefix = "https://images.sprioc.xyz/" + location + "/"
	var resourceBaseURL = prefix + shortcode
	return model.ImgSource{
		Raw:    resourceBaseURL,
		Large:  resourceBaseURL + "?ixlib=rb-0.3.5&q=80&fm=jpg&crop=entropy",
		Medium: resourceBaseURL + "?ixlib=rb-0.3.5&q=80&fm=jpg&crop=entropy&w=1080&fit=max",
		Small:  resourceBaseURL + "?ixlib=rb-0.3.5&q=80&fm=jpg&crop=entropy&w=400&fit=max",
		Thumb:  resourceBaseURL + "?ixlib=rb-0.3.5&q=80&fm=jpg&crop=entropy&w=200&fit=max",
	}
}

// TODO Implement avatar upload
func setAvatar(user model.Ref, source model.ImgSource) error {
	return errors.New("NOT IMPLEMENTED")
}
