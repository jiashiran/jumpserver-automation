package store

import (
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"jumpserver-automation/logs"
	"time"
)

var (
	db         *bolt.DB
	DB         = "/usr/local/db/store.db"
	Bucket     = []byte("StoreBucket")
	ArgsBucket = []byte("ArgsStoreBucket")
	TestBucket = []byte("TestStoreBucket")
)

func init() {
	var err error
	db, err = bolt.Open(DB, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		logs.Logger.Error("open database error:", err)
	}
	if err != nil {
		logs.Logger.Error(err)
	}
	logs.Logger.Info("create database")
	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(Bucket)
		if err != nil {
			fmt.Println("CreateBucket ", Bucket, "err:", err)
		}
		return err
	})
	db.Sync()
	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(ArgsBucket)
		if err != nil {
			fmt.Println("CreateBucket ", ArgsBucket, "err:", err)
		}
		return err
	})
	db.Sync()
	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(TestBucket)
		if err != nil {
			fmt.Println("CreateBucket ", TestBucket, "err:", err)
		}
		return err
	})
	db.Sync()
}

func Update(key string, value string) {
	db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(Bucket)
		err = b.Put([]byte(key), []byte(value))
		return err
	})
	db.Sync()
}

func UpdateArgs(key string, value string) {
	db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(ArgsBucket)
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
		b := tx.Bucket(Bucket)
		bd = b.Get([]byte(key))
		return e
	})
	db.Sync()
	return string(bd)
}

func SelectArgs(key string) string {
	var bd []byte
	db.Update(func(tx *bolt.Tx) error {
		var e error
		defer func() {
			if err := recover(); err != nil {
				e = errors.New(fmt.Sprint(err))
			}
		}()
		b := tx.Bucket(ArgsBucket)
		bd = b.Get([]byte(key))
		return e
	})
	db.Sync()
	return string(bd)
}

func Delete(key string) error {
	var e error
	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(Bucket)
		err := b.Delete([]byte(key))
		e = err
		return e
	})
	return e
}

func DeleteArgs(key string) error {
	var e error
	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(ArgsBucket)
		err := b.Delete([]byte(key))
		e = err
		return e
	})
	return e
}

func SelectAll() map[string]string {
	m := make(map[string]string)
	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(Bucket)
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

func UpdateWithBucket(key string, value string, bucket []byte) {
	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(Bucket)
		err := b.Put([]byte(key), []byte(value))
		return err
	})
	db.Sync()
}

func SelectWithBucket(key string, bucket []byte) string {
	var bd []byte
	db.Update(func(tx *bolt.Tx) error {
		var e error
		defer func() {
			if err := recover(); err != nil {
				e = errors.New(fmt.Sprint(err))
			}
		}()
		b := tx.Bucket(Bucket)
		bd = b.Get([]byte(key))
		return e
	})
	db.Sync()
	return string(bd)
}

func DeleteWithBucket(key string, bucket []byte) error {
	var e error
	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(Bucket)
		err := b.Delete([]byte(key))
		e = err
		return e
	})
	return e
}

func SelectAllWithBucket(bucket []byte) map[string]string {
	m := make(map[string]string)
	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(Bucket)
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
