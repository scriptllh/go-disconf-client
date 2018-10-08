/**
 * @Author: llh
 * @Date:   2018-01-25 16:08:29
 * @Last Modified by:   llh
 */

package disconf_client

import (
	"github.com/tietang/props/kvs"
	"strings"
	"io/ioutil"
	"reflect"
	"fmt"
	"strconv"
)

type Store struct {
	conf interface{}
}

const (
	FILE_PROPERTIES   = ".properties"
	DISCONF_TYPE_FILE = 0
	DISCONF_TYPE_ITEM = 1
	COMMA_STRING      = ","
	CONF_TAG          = "conf"
	AUTO_TAG          = "auto"
	AUTO_TRUE         = "true"
	INIT_CONF         = "initConf"
	AUTO_CONF         = "autoConf"
	STRING_STR        = "string"
	INT64_STR         = "int64"
	INT_STR           = "int"
	FLOAT32_STR       = "float32"
	BOOL_STR          = "bool"
	FLOAT64_STR       = "float64"
	ERR_TYPE_VALUE    = "unknown type"
)

func (s *Store) loadPropertiesDir(filePath string, ignore string) error {
	files, err := ioutil.ReadDir(filePath)
	if err != nil {
		return err
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if ContainString(ignore, file.Name()) {
			continue
		}
		if _, err = s.loadProperties(filePath, file.Name(), INIT_CONF); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) loadProperties(filePath, fileName, flag string, ) (map[string]string, error) {
	ok := strings.HasSuffix(fileName, FILE_PROPERTIES)
	var errs []error
	fileMap := make(map[string]string)
	if ok {
		p, err := kvs.ReadPropertyFile(filePath + fileName)
		if err != nil {
			return nil, err
		}
		fileMap, errs = s.convertProperties(p, flag)
		if len(errs) > 0 {
			return fileMap, fmt.Errorf("convert properties [errs:%v]", errs)
		}
	}
	return fileMap, nil
}

func (s *Store) convertProperties(p *kvs.Properties, flag string) (map[string]string, []error) {
	keys := p.Keys()
	var errs []error
	fileMap := make(map[string]string)
	for _, key := range keys {
		value, err := p.Get(key)
		fileMap[key] = value
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if errs := s.reflectConf(value, key, flag); len(errs) > 0 {
			errs = append(errs, errs...)
		}
	}
	return fileMap, errs
}

func (s *Store) loadItem(key, value, flag string) error {
	if errs := s.reflectConf(value, key, flag); len(errs) > 0 {
		return fmt.Errorf("set conf [err:%v]", errs)
	}
	return nil
}

func (s *Store) loadConf(confs []*Result, filePath string, ignore string) error {
	for _, conf := range confs {
		if ContainString(ignore, conf.Name) {
			continue
		}
		if conf.Genre == DISCONF_TYPE_ITEM {
			if errs := s.reflectConf(conf.Value, conf.Name, INIT_CONF); len(errs) > 0 {
				return fmt.Errorf("set conf [err:%v]", errs)
			}
		}
		if conf.Genre == DISCONF_TYPE_FILE {
			if strings.HasSuffix(conf.Name, FILE_PROPERTIES) {
				var err error
				if _, err = s.loadProperties(filePath, conf.Name, INIT_CONF); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (s *Store) reflectConf(value string, tag string, flag string) []error {
	elems := reflect.TypeOf(s.conf).Elem()
	values := reflect.ValueOf(s.conf).Elem()
	var errs []error
	for i := 0; i < elems.NumField(); i++ {
		if elems.Field(i).Tag.Get(CONF_TAG) == tag {
			switch flag {
			case INIT_CONF:
				if err := s.setConf(elems, values, i, value); err != nil {
					errs = append(errs, err)
				}
			case AUTO_CONF:
				if elems.Field(i).Tag.Get(AUTO_TAG) == AUTO_TRUE {
					if err := s.setConf(elems, values, i, value); err != nil {
						errs = append(errs, err)
					}
				}
			default:
				errs = append(errs, fmt.Errorf("unknown flag"))
			}
		}
	}
	return errs
}

func (s *Store) setConf(elems reflect.Type, values reflect.Value, i int, value string) error {
	switch elems.Field(i).Type.Name() {
	case STRING_STR:
		values.FieldByName(elems.Field(i).Name).SetString(value)
	case INT64_STR:
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		values.FieldByName(elems.Field(i).Name).SetInt(v)
	case INT_STR:
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		values.FieldByName(elems.Field(i).Name).SetInt(v)
	case BOOL_STR:
		v, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		values.FieldByName(elems.Field(i).Name).SetBool(v)
	case FLOAT32_STR:
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		values.FieldByName(elems.Field(i).Name).SetFloat(v)
	case FLOAT64_STR:
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		values.FieldByName(elems.Field(i).Name).SetFloat(v)
	default:
		return fmt.Errorf(ERR_TYPE_VALUE)
	}
	return nil
}

func ContainString(str, content string) bool {
	strs := strings.Split(str, COMMA_STRING)
	for _, s := range strs {
		if s == content {
			return true
		}
	}
	return false
}
