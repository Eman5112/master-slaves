// package main

// import (
// 	"bufio"
// 	"fmt"
// 	"net"
// 	"os"
// 	"os/exec"
// 	"strings"
// 	"time"
// )

// func getHostname() string {
// 	name, err := os.Hostname()
// 	if err != nil {
// 		return "unknown"
// 	}
// 	return name
// }

// func registerWithMaster(masterAddr, hostname, slaveAddr string) error {
// 	conn, err := net.Dial("tcp", masterAddr)
// 	if err != nil {
// 		return fmt.Errorf("failed to connect to master: %v", err)
// 	}
// 	defer conn.Close()

// 	_, err = fmt.Fprintf(conn, "register:%s|%s\n", hostname, slaveAddr)
// 	if err != nil {
// 		return fmt.Errorf("failed to send registration: %v", err)
// 	}
// 	fmt.Println("Registered with master:", masterAddr)
// 	return nil
// }

// func handleConnection(conn net.Conn) {
// 	defer conn.Close()
// 	reader := bufio.NewReader(conn)

// 	for {
// 		command, err := reader.ReadString('\n')
// 		if err != nil {
// 			fmt.Println("Connection closed")
// 			return
// 		}

// 		command = strings.TrimSpace(command)
// 		fmt.Println("Received command:", command)

// 		switch {
// 		case command == "shutdown":
// 			exec.Command("shutdown", "/s", "/t", "5").Run()
// 			conn.Write([]byte("Shutdown started successfully\n"))

// 		case strings.HasPrefix(command, "setbg:"):
// 			path := strings.TrimPrefix(command, "setbg:")
// 			if _, err := os.Stat(path); os.IsNotExist(err) {
// 				conn.Write([]byte("Error: background image not found\n"))
// 				break
// 			}

// 			// Get absolute path using PowerShell
// 			absPath, err := exec.Command("powershell", "-Command", "Resolve-Path '"+path+"' | Select-Object -ExpandProperty Path").Output()
// 			if err != nil {
// 				conn.Write([]byte("Error resolving path\n"))
// 				break
// 			}
// 			cleanPath := strings.TrimSpace(string(absPath))

// 			err = exec.Command("powershell", "-Command", `Add-Type -TypeDefinition @"
// using System.Runtime.InteropServices;
// public class Wallpaper {
// [DllImport("user32.dll",SetLastError=true)]
// public static extern bool SystemParametersInfo(int uAction, int uParam, string lpvParam, int fuWinIni);
// }
// "@; [Wallpaper]::SystemParametersInfo(20, 0, "`+cleanPath+`", 3)`).Run()

// 			if err != nil {
// 				conn.Write([]byte("Failed to set background\n"))
// 			} else {
// 				conn.Write([]byte("Background changed successfully\n"))
// 			}

// 		case command == "ping":
// 			conn.Write([]byte("pong\n"))

// 		case command == "exit":
// 			fmt.Fprintln(conn, "closing")
// 			os.Exit(0)

// 		case command == "sleep":
// 			cmd := exec.Command("rundll32.exe", "powrprof.dll,SetSuspendState", "0", "1", "0")
// 			err := cmd.Run()
// 			if err != nil {
// 				fmt.Fprintln(conn, "error:"+err.Error())
// 			} else {
// 				fmt.Fprintln(conn, "ok")
// 			}

// 		default:
// 			conn.Write([]byte("unknown command\n"))
// 		}
// 	}
// }

// func main() {
// 	hostname := getHostname()
// 	slaveAddr := "127.0.0.1:8081"
// 	masterAddr := "127.0.0.1:9999"

// 	// Retry registration until successful
// 	go func() {
// 		for {
// 			err := registerWithMaster(masterAddr, hostname, slaveAddr)
// 			if err != nil {
// 				fmt.Println("Registration error:", err)
// 				time.Sleep(5 * time.Second)
// 				continue
// 			}
// 			break
// 		}
// 	}()

// 	ln, err := net.Listen("tcp", ":8081")
// 	if err != nil {
// 		fmt.Println("Error starting server:", err)
// 		return
// 	}
// 	defer ln.Close()

// 	fmt.Println("Slave listening on port 8081...")

//		for {
//			conn, err := ln.Accept()
//			if err != nil {
//				fmt.Println("Error accepting connection:", err)
//				continue
//			}
//			go handleConnection(conn)
//		}
//	}
package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

func getHostname() string {
	name, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return name
}

func registerWithMaster(masterAddr, hostname, slaveAddr string) error {
	conn, err := net.Dial("tcp", masterAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to master: %v", err)
	}
	defer conn.Close()

	_, err = fmt.Fprintf(conn, "register:%s|%s\n", hostname, slaveAddr)
	if err != nil {
		return fmt.Errorf("failed to send registration: %v", err)
	}
	fmt.Println("Registered with master:", masterAddr)
	return nil
}

func setWallpaper(path string) {
	ptr, _ := windows.UTF16PtrFromString(path)
	syscall.NewLazyDLL("user32.dll").NewProc("SystemParametersInfoW").Call(
		0x0014, 0, uintptr(unsafe.Pointer(ptr)), 0x01|0x02,
	)
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for {
		command, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Connection closed")
			return
		}

		command = strings.TrimSpace(command)
		fmt.Println("Received command:", command)

		switch {
		case strings.HasPrefix(command, "setbg:"):
			path := strings.TrimPrefix(command, "setbg:")
			// Check if the file exists
			if _, err := os.Stat(path); os.IsNotExist(err) {
				conn.Write([]byte("Error: background image not found\n"))
				break
			}

			// If the file exists, set the wallpaper
			setWallpaper(path)
			conn.Write([]byte("Background changed successfully\n"))

		case command == "ping":
			conn.Write([]byte("pong\n"))

		case command == "exit":
			fmt.Fprintln(conn, "closing")
			os.Exit(0)

		default:
			conn.Write([]byte("unknown command\n"))
		}
	}
}

func main() {
	hostname := getHostname()
	slaveAddr := "127.0.0.1:8081"
	masterAddr := "127.0.0.1:9999"

	// Retry registration until successful
	go func() {
		for {
			err := registerWithMaster(masterAddr, hostname, slaveAddr)
			if err != nil {
				fmt.Println("Registration error:", err)
				time.Sleep(5 * time.Second)
				continue
			}
			break
		}
	}()

	ln, err := net.Listen("tcp", ":8081")
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
	defer ln.Close()

	fmt.Println("Slave listening on port 8081...")

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go handleConnection(conn)
	}
}
