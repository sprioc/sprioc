package model

import (
	"time"

	"gopkg.in/mgo.v2/bson"

	gj "github.com/kpawlik/geojson"
)

// TODO it would be good to have both public and private collections / images.

// TODO need to define external representations of these types and functions
// that map from internal to external.

// Image contains all the proper information for rendering a single photo
type Image struct {
	ID        bson.ObjectId `bson:"_id" json:"-"`
	ShortCode string        `bson:"shortcode" json:"shortcode"`

	MetaData    ImageMetaData `bson:"metadata" json:"metadata"`
	Tags        []string      `bson:"tags,omitempty" json:"tags,omitempty"`
	MachineTags []string      `bson:"machine_tags,omitempty" json:"machine_tags,omitempty"`
	PublishTime time.Time     `bson:"publish_time" json:"publish_time"`

	Owner       DBRef `bson:"owner" json:"-"`
	OwnerExtern User  `bson:"-" json:"owner"`

	Collections     []DBRef  `bson:"collections" json:"-"`
	CollectionLinks []string `bson:"-" json:"collection_links,omitempty"`

	FavoritedBy      []DBRef  `bson:"favorited_by" json:"-"`
	FavoritedByLinks []string `bson:"-" json:"favorited_by_links,omitempty"`

	Sources ImgSource `bson:"sources" json:"sources"`

	Featured  bool `bson:"featured" json:"featured"`
	Downloads int  `bson:"downloads" json:"downloads"`
	Favorites int  `bson:"favorites" json:"favorites"`
	Hidden    bool `bson:"hidden" json:"hidden"`
}

type ImageMetaData struct {
	Aperture       Ratio  `bson:"aperture,omitempty" json:"-"`
	ApertureExtern string `bson:"-" json:"aperture,omitempty"`

	ExposureTime       Ratio  `bson:"exposure_time,omitempty" json:"-"`
	ExposureTimeExtern string `bson:"-" json:"exposure_time,omitempty"`

	FocalLength       Ratio  `bson:"focal_length,omitempty" json:"-"`
	FocalLengthExtern string `bson:"-" json:"focal_length,omitempty"`

	ISO             int        `bson:"iso,omitempty" json:"iso,omitempty"`
	Orientation     string     `bson:"orientation,omitempty" json:"orientation,omitempty"`
	Make            string     `bson:"make,omitempty" json:"make,omitempty"`
	Model           string     `bson:"model,omitempty" json:"model,omitempty"`
	LensMake        string     `bson:"lens_make,omitempty" json:"lens_make,omitempty"`
	LensModel       string     `bson:"lens_model,omitempty" json:"lens_model,omitempty"`
	PixelXDimension int64      `bson:"pixel_xd,omitempty" json:"pixel_xd,omitempty"`
	PixelYDimension int64      `bson:"pixel_yd,omitempty" json:"pixel_yd,omitempty"`
	CaptureTime     time.Time  `bson:"capture_time" json:"capture_time"`
	ImgDirection    float64    `bson:"direction,omitempty" json:"direction,omitempty"`
	Location        gj.Feature `bson:"location,omitempty" json:"location,omitempty"`
}

// ImgSource includes the information about the image itself.
type ImgSource struct {
	Thumb  string `bson:"thumb" json:"thumb"`
	Small  string `bson:"small" json:"small"`
	Medium string `bson:"medium" json:"medium"`
	Large  string `bson:"large" json:"large"`
	Raw    string `bson:"raw" json:"raw"`
}

type User struct {
	ID        bson.ObjectId `bson:"_id" json:"-"`
	ShortCode string        `bson:"shortcode" json:"shortcode"`
	Admin     bool          `bson:"admin" json:"admin"`

	Images     []DBRef  `bson:"images" json:"-"`
	ImageLinks []string `bson:"-" json:"image_links,omitempty"`

	Collections     []DBRef  `bson:"collections" json:"-"`
	CollectionLinks []string `bson:"-" json:"collection_links,omitempty"`

	Followes    []DBRef  `bson:"followes" json:"-"`
	FollowLinks []string `bson:"-" json:"follow_links,omitempty"`

	Favorites     []DBRef  `bson:"favorites" json:"-"`
	FavoriteLinks []string `bson:"-" json:"favorite_links,omitempty"`

	FollowedBy      []DBRef  `bson:"followed_by" json:"-"`
	FollowedByLinks []string `bson:"-" json:"followed_by_links,omitempty"`

	FavoritedBy      []DBRef  `bson:"favorited_by" json:"-"`
	FavoritedByLinks []string `bson:"-" json:"favorited_by_links,omitempty"`

	Email     string     `bson:"email" json:"email"`
	Pass      string     `bson:"password" json:"-"`
	Salt      string     `bson:"salt" json:"-"`
	Name      string     `bson:"name" json:"name,omitempty"`
	Bio       string     `bson:"bio,omitempty" json:"bio,omitempty"`
	URL       string     `bson:"url" json:"url,omitempty"`
	Location  gj.Feature `bson:"loc" json:"loc,omitempty"`
	AvatarURL ImgSource  `bson:"avatar_url" json:"avatar_url"`
}

type Collection struct {
	ID          bson.ObjectId `bson:"_id" json:"-"`
	ShortCode   string        `bson:"shortcode" json:"shortcode"`
	Images      []DBRef       `bson:"images" json:"images,omitempty"`
	Owner       DBRef         `bson:"owner" json:"owner"`
	Contributor []DBRef       `bson:"contributor" json:"contributor"`

	FollowedBy      []DBRef  `bson:"followed_by" json:"-"`
	FollowedByLinks []string `bson:"-" json:"followed_by_links,omitempty"`

	FavoritedBy      []DBRef  `bson:"favorited_by" json:"-"`
	FavoritedByLinks []string `bson:"-" json:"favorited_by_links,omitempty"`

	Desc     string     `bson:"desc,omitempty" json:"desc,omitempty"`
	Title    string     `bson:"title" json:"title"`
	ViewType string     `bson:"view_type" json:"view_type"`
	Location gj.Feature `bson:"location" json:"location"`
}

type DBRef struct {
	Collection string `bson:"collection" json:"collection"`
	Shortcode  string `bson:"shortcode" json:"shortcode"`
	Database   string `bson:"db,omitempty" json:"-"`
}

func (ref DBRef) String() string {
	return getURL(ref)
}
