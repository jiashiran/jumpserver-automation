package store

import (
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"log"
	"time"
)

var (
	db     *bolt.DB
	DB     = "store.db"
	Bucket = []byte("StoreBucket")
)

func init() {
	var err error
	db, err = bolt.Open(DB, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Println("open database error:", err)
	}
	if err != nil {
		log.Println(err)
	}
	log.Println("create database")
}

func Update(key string, value string) {
	db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(Bucket)
		err = b.Put([]byte(key), []byte(value))
		return err
	})
	db.Sync()
}

func Select(key string) string {
	var bd []byte
	db.Update(func(tx *bolt.Tx) error {
		var e error
		defer func() {
			if err := recover(); err != nil {
				e = errors.New(fmt.Sprint(err))
			}
		}()
		b, err := tx.CreateBucketIfNotExists(Bucket)
		if err != nil {
			log.Println("Select error:", err)
			return err
		}
		bd = b.Get([]byte(key))
		return e
	})
	db.Sync()
	return string(bd)
}

func Delete(key string) error {
	var e error
	db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(Bucket)
		if err != nil {
			log.Println("Delete error:", err)
			return err
		}
		err = b.Delete([]byte(key))
		e = err
		return e
	})
	return e
}

func SelectAll() map[string]string {
	m := make(map[string]string)
	db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(Bucket)
		if err != nil {
			log.Println("SelectAll error:", err)
			return err
		}

		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			//fmt.Printf("key=%s, value=%s\n", k, v)
			m[string(k)] = string(v)
		}

		return nil
	})
	db.Sync()
	return m
}

func Close() {
	db.Close()
}
