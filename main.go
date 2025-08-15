
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ---------------- Slave Info -------------------

type SlaveInfo struct {
	Hostname string
	Address  string
	LastSeen time.Time
}

var (
	onlineSlaves = make(map[string]SlaveInfo)
	mutex        sync.Mutex
	clients      = make(map[*websocket.Conn]bool)
	clientsMutex sync.Mutex
)

// ------------ Networking Helpers ------------

func sendCommand(address, command string) (string, error) {
	conn, err := net.DialTimeout("tcp", address, 3*time.Second)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	fmt.Fprintf(conn, command+"\n")
	resp, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp), nil
}

func broadcastSlaves() {
	mutex.Lock()
	defer mutex.Unlock()
	for client := range clients {
		for addr, slave := range onlineSlaves {
			message := fmt.Sprintf("register:%s|%s", slave.Hostname, addr)
			err := client.WriteMessage(websocket.TextMessage, []byte(message))
			if err != nil {
				client.Close()
				delete(clients, client)
			}
		}
	}
}

func startRegistrationListener() {
	ln, err := net.Listen("tcp", ":9999")
	if err != nil {
		fmt.Println("Listener error:", err)
		return
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Accept error:", err)
			continue
		}
		go func(c net.Conn) {
			defer c.Close()
			msg, err := bufio.NewReader(c).ReadString('\n')
			if err != nil {
				fmt.Println("Read error:", err)
				return
			}
			msg = strings.TrimSpace(msg)
			parts := strings.Split(msg, "|")
			if len(parts) == 2 && strings.HasPrefix(parts[0], "register:") {
				host := strings.TrimPrefix(parts[0], "register:")
				addr := parts[1]
				fmt.Printf("Received registration: host=%s, addr=%s\n", host, addr)
				mutex.Lock()
				onlineSlaves[addr] = SlaveInfo{Hostname: host, Address: addr, LastSeen: time.Now()}
				mutex.Unlock()
				broadcastSlaves()
			} else {
				fmt.Println("Ignored invalid message:", msg)
			}
		}(conn)
	}
}

func restartSlave() error {
	cmd := exec.Command("cmd", "/C", "start", "F:\\Study\\Distributed DataBase\\Docter's task\\master_slaves\\Slave\\slave.exe")
	return cmd.Start()
}

// ------------ Wallpaper Change Functions ------------

func changeWallpaperImmediately(address, imagePath string) error {
	winPath := strings.ReplaceAll(imagePath, "/", "\\")

	// Primary method using COM object and direct API call
	psCmd := fmt.Sprintf(
		`setbg:$wallpaper = New-Object -ComObject WScript.Shell; `+
			`$wallpaper.RegWrite('HKCU\\Control Panel\\Desktop\\Wallpaper', '%s', 'REG_SZ'); `+
			`$wallpaper.RegWrite('HKCU\\Control Panel\\Desktop\\WallpaperStyle', '10', 'REG_SZ'); `+
			`$signature = '[DllImport(\"user32.dll\", SetLastError=true)] public static extern int SystemParametersInfo(int uiAction, int uiParam, string pvParam, int fWinIni)'; `+
			`$SPI_SETDESKWALLPAPER = 0x0014; `+
			`$SPIF_UPDATEINIFILE = 0x01; `+
			`$type = Add-Type -MemberDefinition $signature -Name WinAPI -Namespace Wallpaper -PassThru; `+
			`$type::SystemParametersInfo($SPI_SETDESKWALLPAPER, 0, '%s', $SPIF_UPDATEINIFILE)`,
		winPath, winPath)

	// Fallback registry method
	regCmd := fmt.Sprintf(
		`setbg:reg add \"HKCU\\Control Panel\\Desktop\" /v Wallpaper /t REG_SZ /d \"%s\" /f && `+
			`reg add \"HKCU\\Control Panel\\Desktop\" /v WallpaperStyle /t REG_SZ /d \"10\" /f && `+
			`RUNDLL32.EXE user32.dll,UpdatePerUserSystemParameters 1, True`,
		winPath)

	// Try primary method
	_, err := sendCommand(address, psCmd)
	if err != nil {
		// Try fallback method
		_, err = sendCommand(address, regCmd)
		if err != nil {
			return fmt.Errorf("both wallpaper change methods failed: %v", err)
		}
	}

	// Verify the change
	verifyCmd := "setbg:(Get-ItemProperty 'HKCU:\\Control Panel\\Desktop').Wallpaper"
	output, err := sendCommand(address, verifyCmd)
	if err != nil {
		return fmt.Errorf("failed to verify wallpaper change: %v", err)
	}

	if !strings.Contains(output, winPath) {
		return fmt.Errorf("wallpaper change verification failed")
	}

	return nil
}

// ------------ WebSocket and HTTP Handlers ------------

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("WebSocket error:", err)
		return
	}
	fmt.Println("New WebSocket client connected")
	clientsMutex.Lock()
	clients[conn] = true
	clientsMutex.Unlock()

	defer func() {
		clientsMutex.Lock()
		delete(clients, conn)
		clientsMutex.Unlock()
		conn.Close()
	}()

	broadcastSlaves()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func handleCommand(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Address string `json:"address"`
		Command string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	result, err := sendCommand(req.Address, req.Command)
	resp := struct {
		Result string `json:"result"`
	}{Result: result}
	if err != nil {
		resp.Result = fmt.Sprintf("Error: %v", err)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleWallpaperChange(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Address   string `json:"address"`
		ImagePath string `json:"imagePath"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := changeWallpaperImmediately(req.Address, req.ImagePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Wallpaper changed successfully"))
}

func handleRestart(w http.ResponseWriter, r *http.Request) {
	err := restartSlave()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte("Restart command sent"))
}

func main() {
	// Serve static files
	http.Handle("/", http.FileServer(http.Dir("web")))
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/command", handleCommand)
	http.HandleFunc("/wallpaper", handleWallpaperChange)
	http.HandleFunc("/restart", handleRestart)

	// Start registration listener
	go startRegistrationListener()

	// Periodic ping to check slave status
	go func() {
		for {
			time.Sleep(5 * time.Second)
			var removed []string
			mutex.Lock()
			for addr := range onlineSlaves {
				if _, err := sendCommand(addr, "ping"); err != nil {
					removed = append(removed, addr)
				}
			}
			for _, addr := range removed {
				delete(onlineSlaves, addr)
			}
			mutex.Unlock()
			if len(removed) > 0 {
				broadcastSlaves()
			}
		}
	}()

	fmt.Println("Server running on http://localhost:8082")
	http.ListenAndServe(":8082", nil)
}

