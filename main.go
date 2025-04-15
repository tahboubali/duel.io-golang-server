package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"math"
	"net/http"
	"sort"
	"sync"
)

const (
	RankRange         = float64(10)
	MatchmakingStatus = 0
	FoundDuelStatus   = 1
)

type Server struct {
	websocket.Upgrader
	mu      sync.Mutex
	wg      sync.WaitGroup
	players map[string]*Player
	conns   map[string]*Conn
}

func NewServer() *Server {
	return &Server{
		Upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		players: make(map[string]*Player),
		conns:   make(map[string]*Conn),
	}
}

func (s *Server) Run() {
	http.HandleFunc("/connect", func(w http.ResponseWriter, r *http.Request) {
		conn, err := s.Upgrade(w, r, nil)
		if err != nil {
			log.Println("failed to upgrade ws connection for:", conn.RemoteAddr().String())
			return
		}
		s.wg.Add(1)
		go s.handleConn(&Conn{Conn: conn})
	})
	log.Println("server started on port :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func (s *Server) handleConn(conn *Conn) {
	defer func() {
		s.wg.Done()
		s.closeConn(conn)
		_ = s.broadcastPlayers()
	}()
	s.mu.Lock()
	s.conns[conn.RemoteAddr().String()] = conn
	s.mu.Unlock()
	log.Println("new connection established:", conn.RemoteAddr())
	for {
		//if conn.player != nil && conn.player.dueling {
		//	continue
		//}
		_, reader, err := conn.NextReader()
		if err != nil {
			log.Println("error getting next reader for connection (" + conn.RemoteAddr().String() + ")")
			return
		}
		var data struct {
			RequestType string         `json:"request_type"`
			Data        map[string]any `json:"data"`
		}
		err = json.NewDecoder(reader).Decode(&data)
		if err != nil {
			log.Println("failed to decode json")
			conn.writeErrMsg("", "Failed to decode JSON")
			continue
		}
		//log.Println("new message received:", data, "from:", *conn)
		switch data.RequestType {
		case "new-player":
			username := data.Data["username"].(string)
			if s.playerExists(username) {
				conn.writeErrMsg("new-player", fmt.Sprintf("Player with username \"%s\" already exists", username))
				break
			}
			if conn.player != nil {
				conn.writeErrMsg("new-player", "Player is already registered on connection")
				break
			}
			if len(username) > 15 {
				conn.writeErrMsg("new-player", "Username must be under 16 characters")
				break
			}
			s.mu.Lock()
			player := NewPlayer(username, conn)
			s.players[username] = player
			conn.player = player
			s.mu.Unlock()

			err := conn.WriteJSON(map[string]any{
				"request_type": "new-player-success",
				"data": map[string]any{
					"username": username,
				},
			})
			if err != nil {
				log.Println(err)
				break
			}
			if s.broadcastPlayers() != nil {
				fmt.Println("error broadcasting players")
			}
		case "enter-duel":
			if conn.player == nil {
				conn.writeErrMsg("enter-duel", "Player is not registered")
				break
			}
			if conn.player.dueling || conn.player.matchmaking {
				conn.writeErrMsg("enter-duel", "Player already entered duel")
				break
			}
			username := conn.player.Username
			_ = conn.WriteJSON(map[string]any{
				"request_type": "enter-duel",
				"message":      "Matchmaking...",
				"status":       MatchmakingStatus,
			})
			s.mu.Lock()
			s.players[username].matchmaking = true
			s.mu.Unlock()
			go s.enterDuel(username)
		case "sign-out":
			if conn.player == nil {
				conn.writeErrMsg("sign-out", fmt.Sprintf("Player is not registered"))
				break
			}
			username := conn.player.Username
			conn.player.duelChan <- &DuelMessage{
				Type: "sign-out",
			}
			s.mu.Lock()
			conn.player.dueling = false
			conn.player.matchmaking = false
			delete(s.players, username)
			conn.player = nil
			s.mu.Unlock()
			_ = s.broadcastPlayers()
		case "game-state":
			if conn.player == nil {
				conn.writeErrMsg("game-state", fmt.Sprintf("Player is not registered"))
				break
			}
			if !conn.player.dueling {
				conn.writeErrMsg("game-state", fmt.Sprintf("Player is not dueling"))
				break
			}
			conn.player.duelChan <- &DuelMessage{
				Type: "game-state",
				Data: data.Data,
			}
		case "game-end":
			if conn.player == nil {
				conn.writeErrMsg("game-end", fmt.Sprintf("Player is not registered"))
				break
			}
			if !conn.player.dueling {
				conn.writeErrMsg("game-end", fmt.Sprintf("Player is not dueling"))
				break
			}
			conn.player.duelChan <- &DuelMessage{
				Type: "game-end",
				Data: data.Data,
			}
		}
	}
}

func (s *Server) closeConn(conn *Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if conn.player != nil {
		delete(s.players, conn.player.Username)
	}
	delete(s.conns, conn.RemoteAddr().String())
}

func (s *Server) enterDuel(username string) {
	s.mu.Lock()
	player := s.players[username]
	s.mu.Unlock()
	minDiff := math.MaxFloat64
	var match *Player
	for match == nil {
		if player.dueling {
			return
		}
		s.mu.Lock()
		if _, exists := s.players[username]; !exists {
			s.mu.Unlock()
			return
		}
		for _, curr := range s.players {
			if curr != player && !curr.dueling && curr.matchmaking {
				diff := math.Abs(player.Rank - curr.Rank)
				if diff < minDiff {
					minDiff = diff
					match = curr
				}
			}
		}
		s.mu.Unlock()
	}
	s.duel(player, match)
}

func (s *Server) duel(p1 *Player, p2 *Player) {
	if p1.dueling && p2.dueling {
		return
	}
	s.mu.Lock()
	p1.sendEnterDuelSuccess(p2, "left")
	p2.sendEnterDuelSuccess(p1, "right")
	p1.matchmaking = false
	p2.matchmaking = false
	p1.dueling = true
	p2.dueling = true
	s.mu.Unlock()
	p1Chan := make(chan *DuelMessage)
	p2Chan := make(chan *DuelMessage)
	log.Println(p1.Username, "is dueling", p2.Username)
	wg := sync.WaitGroup{}
	wg.Add(2)
	go p1.duelPlayer(p2, p1Chan, p2Chan, &wg, &s.mu)
	go p2.duelPlayer(p1, p2Chan, p1Chan, &wg, &s.mu)
	wg.Wait()
	//close(p1Chan)
	//close(p2Chan)
	fmt.Println("finished duel!")
	_ = s.broadcastPlayers()
}

func (p *Player) sendEnterDuelSuccess(other *Player, pos string) {
	_ = p.conn.WriteJSON(map[string]any{
		"request_type": "enter-duel",
		"message":      fmt.Sprintf("Successfully started duel with \"%s\"", other.Username),
		"status":       FoundDuelStatus,
		"position":     pos,
		"match": map[string]any{
			"username": other.Username,
			"rank":     other.Rank,
		},
	})
}

func (p *Player) duelPlayer(other *Player, playerChan, otherChan chan *DuelMessage, wg *sync.WaitGroup, mu *sync.Mutex) {
	defer func() {
		p.dueling = false
		wg.Done()
	}()
	go func() {
		for p.dueling && other.dueling {
			msg := <-p.duelChan
			if msg.Type == "sign-out" {
				gameEndMessage := &DuelMessage{
					Type: "game-end",
					Data: map[string]any{
						"player_won": other.Username,
					},
				}
				playerChan <- gameEndMessage
				otherChan <- gameEndMessage
				return
			}
			if msg.Type == "game-end" {
				playerChan <- msg
				otherChan <- msg
				return
			}
			playerChan <- msg
		}
	}()

	for p.dueling && other.dueling {
		select {
		case otherMsg := <-otherChan:
			switch otherMsg.Type {
			case "game-end":
				mu.Lock()
				p.sendDuelResults(otherMsg, other)
				mu.Unlock()
				return
			case "game-state":
				mu.Lock()
				err := p.conn.WriteJSON(map[string]any{
					"request_type": otherMsg.Type,
					"data":         otherMsg.Data,
				})
				mu.Unlock()
				if err != nil {
					log.Println("write err:", err)
				}
			}
		}
	}
}

func (p *Player) sendDuelResults(msg *DuelMessage, other *Player) {
	var playerWon *Player
	if msg.Data["player_won"] == p.Username {
		p.Rank++
		playerWon = p
	} else {
		p.Rank--
		playerWon = other
	}
	_ = p.conn.WriteJSON(map[string]any{
		"request_type": msg.Type,
		"result":       playerWon.Username,
		"new_rank":     p.Rank,
	})
}

func (s *Server) broadcastJSON(msg any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, player := range s.players {
		err := player.conn.WriteJSON(msg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) broadcastPlayers() error {
	players := make([]*Player, 0, len(s.players))
	for _, player := range s.players {
		players = append(players, player)
	}
	sort.Slice(players, func(i, j int) bool {
		return players[i].Rank > players[j].Rank
	})
	err := s.broadcastJSON(map[string]any{
		"request_type": "players-update",
		"players":      players,
	})
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func (s *Server) playerExists(username string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, exists := s.players[username]
	return exists
}

type Conn struct {
	*websocket.Conn
	player *Player
}

func (c *Conn) writeErrMsg(requestType, msg string) {
	errType := "error"
	if requestType != "" {
		errType = requestType + "-error"
	}
	err := c.WriteJSON(map[string]any{
		"request_type": errType,
		"message":      msg,
	})
	if err != nil {
		log.Println("failed to write error message:", msg, "to:")
	}
}

type Player struct {
	Username    string  `json:"username"`
	Rank        float64 `json:"rank"`
	dueling     bool
	matchmaking bool
	conn        *Conn
	duelChan    chan *DuelMessage
	*GameState
}

func NewPlayer(username string, conn *Conn) *Player {
	return &Player{
		conn:     conn,
		Username: username,
		Rank:     RankRange / 2,
		duelChan: make(chan *DuelMessage, 1),
	}
}

type GameState struct {
	Pos     *Pos      `json:"position"`
	Bullets []*Bullet `json:"bullets"`
	Blocks  []*Block  `json:"blocks"`
}

type DuelMessage struct {
	Type string         `json:"request_type"`
	Data map[string]any `json:"data"`
}

type Bullet struct {
	Pos   *Pos    `json:"position"`
	Angle float64 `json:"angle"`
}

type Block struct {
	*Pos
	*Size
}

type Pos struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Size struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

func main() {
	NewServer().Run()
}
