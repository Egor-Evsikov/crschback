package db

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestDB(t *testing.T) {

	conf, _ := LoadDBConfig("./dbConfig.yaml")
	d, _ := ConnDB(conf)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	var uid int
	func() {
		uid, err := d.GetUserId(ctx, "asdf")
		if err != nil {
			t.Error("Ошибка поиска id пользователя ", uid, err)
		}
	}()
	fmt.Println(uid)

	func() {
		id, err := d.MakeDirectory(ctx, "qwqw", "asdf")
		fmt.Println("id = ", id)
		if err != nil {
			t.Error("Ошибка создания директории ", err)
		}
	}()

	func() {
		ok, err := d.CheckDir(ctx, "aaa", "zzz")
		fmt.Println("exist = ", ok)
		if err != nil {
			t.Error("Ошибка проверка директории ", err)
		}
	}()

	func() {
		dirs, err := d.GetDirectories(ctx, "zzz")
		fmt.Println(dirs)
		if err != nil {
			t.Error("Ошибка получения директорий ", err)
		}
	}()

	// func() {
	// 	err := d.DeleteUser(ctx, "asdf", "asdf")
	// 	if err != nil {
	// 		t.Error(" Ошибка удаления пользователя ", err)
	// 	}
	// }()

	// func() {
	// 	_, err := d.MakeDirectory(ctx, "qwerty", "test")
	// 	if err != nil {
	// 		t.Error("Директория не создана ", err)
	// 	}

	// }()

	// func() {
	// 	err := d.DeleteDirectory(ctx, "qwerty", "test")
	// 	if err != nil {
	// 		t.Error(" Ошибка удаления директории ", err)
	// 	}
	// }()

	// func() {
	// 	err := d.AddUserToDir(ctx, "test", "qwerty", "aaa")
	// 	if err != nil {
	// 		t.Error(" Ошибка добавления пользователя к директории ", err)
	// 	}
	// }()
}
