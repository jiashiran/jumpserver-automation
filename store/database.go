package store

import (
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"log"
	"time"
)

var db *bolt.DB

func init() {
	var err error
	db, err = bolt.Open("my.db", 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Println("open database error:", err)
	}
}

func Update(key string, value string) {
	db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("MyBucket"))
		err = b.Put([]byte(key), []byte(value))
		return err
	})
}

func Select(key string) string {
	var bd []byte
	db.View(func(tx *bolt.Tx) error {
		var e error
		defer func() {
			if err := recover(); err != nil {
				e = errors.New(fmt.Sprint(err))
			}
		}()
		b := tx.Bucket([]byte("MyBucket"))
		bd = b.Get([]byte(key))
		return e
	})
	return string(bd)
}

func Delete(key string) error {
	var e error
	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("MyBucket"))
		err := b.Delete([]byte(key))
		e = err
		return e
	})
	return e
}

func SelectAll() map[string]string {
	m := make(map[string]string)
	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("MyBucket"))

		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			fmt.Printf("key=%s, value=%s\n", k, v)
			m[string(k)] = string(v)
		}

		return nil
	})

	return m
}

func Close() {
	db.Close()
}
