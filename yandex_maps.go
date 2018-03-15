package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type YandexLocations struct {
	Response struct {
		GeoObjectCollection struct {
			MetaDataProperty struct {
				GeocoderResponseMetaData struct {
					Request string `json:"request"`
					Found   string `json:"found"`
					Results string `json:"results"`
				} `json:"GeocoderResponseMetaData"`
			} `json:"metaDataProperty"`
			FeatureMember []struct {
				GeoObject struct {
					MetaDataProperty struct {
						GeocoderMetaData struct {
							Kind      string `json:"kind"`
							Text      string `json:"text"`
							Precision string `json:"precision"`
							Address   struct {
								CountryCode string `json:"country_code"`
								Formatted   string `json:"formatted"`
								Components  []struct {
									Kind string `json:"kind"`
									Name string `json:"name"`
								} `json:"Components"`
							} `json:"Address"`
							AddressDetails struct {
								Country struct {
									AddressLine        string `json:"AddressLine"`
									CountryNameCode    string `json:"CountryNameCode"`
									CountryName        string `json:"CountryName"`
									AdministrativeArea struct {
										AdministrativeAreaName string `json:"AdministrativeAreaName"`
										Locality               struct {
											LocalityName string `json:"LocalityName"`
											Thoroughfare struct {
												ThoroughfareName string `json:"ThoroughfareName"`
												Premise          struct {
													PremiseName string `json:"PremiseName"`
												} `json:"Premise"`
											} `json:"Thoroughfare"`
										} `json:"Locality"`
										SubAdministrativeArea struct {
											SubAdministrativeAreaName string `json:"SubAdministrativeAreaName"`
											Locality                  struct {
												LocalityName string `json:"LocalityName"`
											} `json:"Locality"`
										} `json:"SubAdministrativeArea"`
									} `json:"AdministrativeArea"`
								} `json:"Country"`
							} `json:"AddressDetails"`
						} `json:"GeocoderMetaData"`
					} `json:"metaDataProperty"`
					Description string `json:"description"`
					Name        string `json:"name"`
					BoundedBy   struct {
						Envelope struct {
							LowerCorner string `json:"lowerCorner"`
							UpperCorner string `json:"upperCorner"`
						} `json:"Envelope"`
					} `json:"boundedBy"`
					Point struct {
						Pos string `json:"pos"`
					} `json:"Point"`
				} `json:"GeoObject"`
			} `json:"featureMember"`
		} `json:"GeoObjectCollection"`
	} `json:"response"`
}

const YANDEX_REQUEST_TEMPLATE = "https://geocode-maps.yandex.ru/1.x/?format=json&geocode=%s"

var UnknownLocationError = errors.New("unknown location")

func GetUserLocation(phrase string) (*Location, error) {
	resp, err := http.Get(fmt.Sprintf(YANDEX_REQUEST_TEMPLATE, url.QueryEscape(phrase)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var yandexLocs YandexLocations
	err = json.NewDecoder(resp.Body).Decode(&yandexLocs)
	if err != nil {
		return nil, err
	}

	var city string
	// we should always try to find a nearest subway station
	// if there is no subway stations, find the first street in the city and return city
	if len(yandexLocs.Response.GeoObjectCollection.FeatureMember) == 0 {
		return nil, UnknownLocationError
	}
	for _, member := range yandexLocs.Response.GeoObjectCollection.FeatureMember {
		kind := member.GeoObject.MetaDataProperty.GeocoderMetaData.Kind
		if kind == "metro" {
			subway := strings.TrimSpace(strings.Replace(member.GeoObject.Name, "метро", "", -1))
			area := member.GeoObject.MetaDataProperty.GeocoderMetaData.AddressDetails.Country.AdministrativeArea
			if area.SubAdministrativeArea.Locality.LocalityName == "" {
				city = area.Locality.LocalityName
			} else {
				city = area.SubAdministrativeArea.Locality.LocalityName
			}
			return &Location{
				City:   city,
				Subway: subway,
			}, nil
		} else if kind == "street" {
			area := member.GeoObject.MetaDataProperty.GeocoderMetaData.AddressDetails.Country.AdministrativeArea
			if area.SubAdministrativeArea.Locality.LocalityName == "" {
				city = area.Locality.LocalityName
			} else {
				city = area.SubAdministrativeArea.Locality.LocalityName
			}
		}
	}

	if city != "" {
		return &Location{City: city}, nil
	}
	return nil, UnknownLocationError
}
