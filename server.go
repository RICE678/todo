package main

import (
	"bufio"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"net"
	"strconv"
	"strings"
	"sync"
)

var db1 *sql.DB
var conns []net.Conn
var connmutex sync.Mutex
var some map[net.Conn]string

func initDB1() (err error) {
	dsn := "root:123456@tcp(192.168.1.111:3306)/test"
	db1, err = sql.Open("mysql", dsn)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = db1.Ping()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("连接成功")
	db1.SetMaxIdleConns(100)
	return
}

func main() {
	listen, err := net.Listen("tcp", "192.168.1.111:20000")
	if err != nil {
		fmt.Println("listen failed,err:", err)
		return
	}
	defer listen.Close()
	some = make(map[net.Conn]string)
	initDB1()
	fmt.Println("聊天室启动成功！")
	for {
		conn, err := listen.Accept()
		if err != nil {
			fmt.Println("accecpt failed,err:", err)
			continue
		}
		connmutex.Lock()
		conns = append(conns, conn)
		connmutex.Unlock()
		go list(conn)
	}
}
func list(conn net.Conn) {
	reader := bufio.NewReader(conn)
	msg, _ := reader.ReadString('\n')
	msg = strings.TrimSpace(msg)
	some[conn] = msg
	broadcast(msg + " 上线啦！欢迎欢迎~~\n")
	broadcast("当前人数：" + strconv.Itoa(len(conns)) + "\n")
	defer func() {
		connmutex.Lock()
		for i, c := range conns {
			if c == conn {
				conns = append(conns[:i], conns[i+1:]...)
				break
			}
		}
		connmutex.Unlock()
		broadcast(some[conn] + " 下线啦~" + "\n")
		delete(some, conn)
		broadcast("当前人数：" + strconv.Itoa(len(conns)) + "\n")
		conn.Close()
	}()
	reader = bufio.NewReader(conn)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("客户端断开连接")
			sqlStr := `UPDATE user SET status=? WHERE name = ?;`
			_, _ = db1.Exec(sqlStr, 0, some[conn])
			return
		}
		msg = strings.TrimSpace(msg)
		if msg == "@" {
			broadcast2(conn)
			continue
		}
		if msg == "\\file" {
			broadcast3(conn)
			continue
		}
		if msg == "\\rename" {
			channame(conn)
			continue
		}
		broadcast(msg)
	}
}
func broadcast2(conn net.Conn) {
	reader := bufio.NewReader(conn)
	name, err := reader.ReadString('\n')
	name = strings.TrimSpace(name)
	if err != nil {
		return
	}
	fname, _ := reader.ReadString('\n')
	fname = strings.TrimSpace(fname)

	for {
		if quest(name) == 0 {
			conn.Write([]byte(">>> " + name + "已下线，已自动退出私聊模式\n"))
			return
		}
		msg, err := reader.ReadString('\n')
		msg = strings.TrimSpace(msg)
		if msg == "#" {
			return
		}
		if err != nil {
			fmt.Println("err：", err)
			return
		}
		found := false

		connmutex.Lock()
		for _, c := range conns {
			if some[c] == name {
				c.Write([]byte("【私聊from：" + fname + "】" + msg + "\n"))
				found = true
				break
			}
		}
		connmutex.Unlock()

		if !found {
			conn.Write([]byte(">>> " + name + "已下线，已自动退出私聊模式\n"))
			return
		}
	}
}
func broadcast(msg string) {
	connmutex.Lock()
	defer connmutex.Unlock()
	for _, c := range conns {
		if !strings.HasSuffix(msg, "\n") {
			msg += "\n"
		}
		c.Write([]byte(msg))
	}
}
func broadcast3(conn net.Conn) {
	scanner := bufio.NewReader(conn)
	filename, _ := scanner.ReadString('\n')
	filename = strings.TrimSpace(filename)
	connmutex.Lock()

	for _, c := range conns {
		if c != conn {
			c.Write([]byte("\\file\n"))
			c.Write([]byte(filename + "\n"))
		}
	}
	connmutex.Unlock()
	for {
		msg, _ := scanner.ReadString('\n')
		msg = strings.TrimSpace(msg)
		connmutex.Lock()
		for _, c := range conns {
			if c != conn {
				c.Write([]byte(msg + "\n"))
			}
		}
		connmutex.Unlock()
		if msg == "%%" {
			return
		}
	}

}
func quest(name string) int {
	sqlStr := `select status from user where name=?;`
	row := db1.QueryRow(sqlStr, name)
	var s int
	row.Scan(&s)
	if s == 0 {
		return 0
	}
	return 1
}

func channame(conn net.Conn) {
	reader := bufio.NewReader(conn)
	msg, _ := reader.ReadString('\n')
	msg = strings.TrimSpace(msg)
	old := some[conn]
	some[conn] = msg
	broadcast(old + " 改名为 " + msg + "\n")
}
