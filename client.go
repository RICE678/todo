package main

import (
	"bufio"
	"database/sql"
	"encoding/base64"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

var db *sql.DB
var id int64
var climutex sync.Mutex

type user struct {
	name   string
	id     int64
	key    string
	status int64
	conn   net.Conn
}

var u1 user

func initDB() (err error) {
	dsn := "root:123456@tcp(192.168.1.111:3306)/test"
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		return
	}
	err = db.Ping()
	if err != nil {
		return
	}
	fmt.Println("连接成功")
	db.SetMaxIdleConns(100)
	return
}
func main() {
	err := initDB()
	if err != nil {
		fmt.Println("连接失败！~")
	}
	for {
		show()
		fmt.Println("1、登录")
		fmt.Println("2、注册")
		fmt.Println("0、退出")
		fmt.Printf("请输入你的选择：")

		scanner := bufio.NewReader(os.Stdin)
		n, _ := scanner.ReadString('\n')
		n = strings.TrimSpace(n)
		if n == "0" {
			fmt.Println("下次再见~")
			return
		} else if n == "1" {
			update()
		} else if n == "2" {
			insert()
		} else {
			fmt.Println("输入有误，请重试！")
		}
	}
}
func insert() {
	fmt.Println("注册")
	fmt.Printf("请输入用户名：")
	scanner := bufio.NewReader(os.Stdin)
	name, _ := scanner.ReadString('\n')
	name = strings.TrimSpace(name)
	if searchname2(name) == 0 {
		return
	}
	fmt.Printf("请输入密码：")
	key, _ := scanner.ReadString('\n')
	key = strings.TrimSpace(key)

	sqlStr := `insert into user(name,user_key) values(?,?);`
	ret, err := db.Exec(sqlStr, name, key)
	if err != nil {
		fmt.Println("注册失败！请重试~")
		return
	}
	id, _ = ret.LastInsertId()
	fmt.Println("注册成功！请即刻登录~")
	update()
}
func update() {
	fmt.Println("登录")
	fmt.Printf("请输入用户名：")
	scanner := bufio.NewReader(os.Stdin)
	name, _ := scanner.ReadString('\n')
	name = strings.TrimSpace(name)
	if searchname1(name) == 0 {
		return
	}
	sqlStr := `select status from user where name=?;`
	ret := db.QueryRow(sqlStr, name)
	var s user
	ret.Scan(&s.status)
	if s.status == 1 {
		fmt.Println("您已登录，不能重复登录哦~~")
		return
	}
	fmt.Printf("请输入密码：")
	key, _ := scanner.ReadString('\n')
	key = strings.TrimSpace(key)
	sqlStr = "select user_key,id from user where name=?;"
	row := db.QueryRow(sqlStr, name)
	row.Scan(&u1.key, &u1.id)
	if u1.key == key {
		u1.name = name
		t := time.Now()
		s1 := t.Format("2006年1月2日 15:04:05") //123456
		fmt.Println("\t" + s1)
		fmt.Println("\t登录成功，欢迎回来！~")
		progress()
	} else {
		fmt.Println("密码错误，请重试！")
		return
	}
}

var f bool = false
var f1 bool = false

func progress() {
	conn, err := net.Dial("tcp", "192.168.1.111:20000")
	if err != nil {
		fmt.Println("err:", err)
		return
	}
	defer conn.Close()
	fmt.Fprintln(conn, u1.name)
	go func() {
		reader := bufio.NewReader(conn)
	po:
		for {
			input, err := reader.ReadString('\n')
			if err != nil && f == false {
				fmt.Println("连接服务器失败")
				return
			}
			climutex.Lock()
			input = strings.TrimSpace(input)
			fmt.Print("\r\033[k") //回到当前行开头，清除从光标到行尾的内容
			if input == "\\file" {
				climutex.Unlock()
				filename1, _ := reader.ReadString('\n')
				filename1 = strings.TrimSpace(filename1)
				file1, err := os.Create(filename1 + "副本")
				if err != nil {
					fmt.Println("创建失败！", err)
					continue
				}
				for {
					text, _ := reader.ReadString('\n')
					text = strings.TrimSpace(text)
					if text == "%%" {
						file1.Close()
						climutex.Lock()
						fmt.Printf(">>> 文件%s接收完成\n输入：", filename1)
						climutex.Unlock()
						goto po
					}
					decoded, err := base64.StdEncoding.DecodeString(text)
					if err != nil {
						fmt.Println("文件解码失败：", err)
						continue
					}
					file1.Write(decoded)
				}
			} else if strings.Contains(input, u1.name+" 下线啦~") {
				fmt.Print(">>> " + input + "\n")
			} else if f1 == true {
				fmt.Print(">>> " + input + "\n")
				fmt.Print("\r\033[k【私聊】你说：")
			} else {
				fmt.Print(">>> " + input + "\n输入:")
			}
			climutex.Unlock()
		}
	}()

	sqlStr := `update user set status=? where id=?;`
	_, err = db.Exec(sqlStr, 1, u1.id)
	if err != nil {
		fmt.Println("err:", err)
		return
	}
	scanner := bufio.NewReader(os.Stdin)
	for {
		text, _ := scanner.ReadString('\n')
		text = strings.TrimSpace(text)
		if text == "\\q" {
			fmt.Println("退出聊天啦~")
			fmt.Print("\r\033[k")
			u1.status = 0
			_, err = fmt.Fprintln(conn, u1.name+" 下线啦~")
			if err != nil {
				fmt.Println("err:", err)
			}
			sqlStr := `update user set status=? where id=?;`
			_, err := db.Exec(sqlStr, 0, u1.id)
			if err != nil {
				fmt.Println("err:", err)
			}
			f = true
			conn.Close()
			os.Exit(0)
		} else if text == "\\cn" {
			fmt.Printf("原昵称：%s\n", u1.name)
			fmt.Printf("你想修改成:")
			scanner := bufio.NewReader(os.Stdin)
			text, _ := scanner.ReadString('\n')
			text = strings.TrimSpace(text)
			if searchname2(text) == 0 {
				continue
			}
			sqlStr := `update user set name=? where id=?;`
			ret, err := db.Exec(sqlStr, text, u1.id)
			if err != nil {
				fmt.Println("改名失败，原因是：", err)
				return
			}
			ret.LastInsertId()
			fmt.Println("改名成功！~")
			fmt.Print("\r\033[k输入：")
			u1.name = text
			fmt.Fprintln(conn, "\\rename")
			fmt.Fprintln(conn, u1.name)
		} else if text == "\\ck" {
			chankey(u1)
		} else if text == "@" {
		retry:
			fmt.Println("请输入你想私聊的人：(输入\\back退出)")
			name, _ := scanner.ReadString('\n')
			name = strings.TrimSpace(name)
			if name == "\\back" {
				continue
			}
			if u1.name == name {
				fmt.Println("暂时不能和自己私聊哦~")
				goto retry
			}
			if searchname1(name) == 0 {
				goto retry
			}
			sqlStr = `SELECT status FROM user where name=?;`
			row := db.QueryRow(sqlStr, name)
			var s user
			row.Scan(&s.status)
			if s.status == 1 {
				fmt.Println("已开启对 " + name + " 的私聊模式！按'#'进入公聊")
				fmt.Fprintln(conn, "@")
				fmt.Fprintln(conn, name)
				fmt.Fprintln(conn, u1.name)
				f1 = true
				for {
					fmt.Printf("【私聊】你说：")
					text, _ := scanner.ReadString('\n')
					text = strings.TrimSpace(text)
					if text == "#" {
						fmt.Println("您已退出私聊！~~~")
						fmt.Fprintln(conn, "#")
						fmt.Print("\r\033[k输入：")
						f1 = false
						break
					}
					fmt.Fprintln(conn, text)
				}
			} else {
				fmt.Println(name + "暂未在线~")
				goto retry
			}
		} else if text == "\\file" {
			fmt.Println("请输入文件的路径：")
			filename, _ := scanner.ReadString('\n')
			filename = strings.TrimSpace(filename)
			file, err := os.Open(filename)
			if err != nil {
				fmt.Println("读取文件失败，", err)
				continue
			}
			defer file.Close()
			fmt.Fprintln(conn, "\\file")
			fmt.Fprintln(conn, filename)
			bs := make([]byte, 1024)
			n := -1
			for {
				n, err = file.Read(bs)
				if n == 0 || err == io.EOF {
					fmt.Println("发送成功！")
					fmt.Fprintln(conn, "%%")
					fmt.Print("\r\033[k输入：")
					break
				}
				encoded := base64.StdEncoding.EncodeToString(bs[:n])
				fmt.Fprintln(conn, encoded)
			}
			continue
		} else if text == "\\help" {
			show()
		} else {
			_, err := fmt.Fprintln(conn, u1.name+"："+text)
			if err != nil {
				fmt.Println("发送信息失败")
				continue
			}
		}
	}
}

func chankey(u user) {
	fmt.Printf("请输入原密码以确保安全：")
	scanner := bufio.NewReader(os.Stdin)
	text, _ := scanner.ReadString('\n')
	text = strings.TrimSpace(text)
	if text != u.key {
		fmt.Println("密码输入错误！")
		return
	}
	fmt.Printf("请输入新密码：")
	p, _ := scanner.ReadString('\n')
	p = strings.TrimSpace(p)
	sqlStr := `update user set user_key=? where id=?;`
	ret, err := db.Exec(sqlStr, p, u.id)
	if err != nil {
		fmt.Println("密码修改失败，原因是：", err)
		return
	}
	ret.LastInsertId()
	fmt.Println("密码修改成功！")
	fmt.Print("\r\033[k输入：")
	u1.key = p
}
func show() {
	fmt.Printf("\t    公  告\n\t欢迎来到聊天室！\n\t-输入'\\q'为退出\n\t-输入'@'为公聊转成私聊\n\t-输入'#'为私聊转为公聊\n\t-输入'\\file'为发送文件\n\t-输入'\\cn'为修改昵称\n\t-输入'\\ck'为修改密码\n\t-输入'\\help'为查看说明\n\t祝您聊的开心~~\n")
}

func searchname1(name string) int {
	sqlStr := `SELECT id FROM user WHERE name=?;`
	row := db.QueryRow(sqlStr, name)
	var id int64
	err := row.Scan(&id)
	if err == sql.ErrNoRows {
		fmt.Println(name + "用户不存在，请检查用户名！")
		return 0
	}
	return 1
}
func searchname2(name string) int {
	sqlStr := `SELECT id FROM user WHERE name=?;`
	row := db.QueryRow(sqlStr, name)
	var id int64
	err := row.Scan(&id)
	if err != sql.ErrNoRows {
		fmt.Println(name + "用户名已存在！")
		return 0
	}
	return 1
}
