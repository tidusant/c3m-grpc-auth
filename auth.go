package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/tidusant/c3m-common/log"

	pb "github.com/tidusant/c3m-grpc-protoc/protoc"
	rpch "github.com/tidusant/chadmin-repo/cuahang"
	"github.com/tidusant/chadmin-repo/models"

	"context"
	"google.golang.org/grpc"
)

const (
	name string = "auth"
	ver  string = "1"
)

type service struct {
	pb.UnimplementedGRPCServicesServer
}

func (s *service) Call(ctx context.Context, in *pb.RPCRequest) (*pb.RPCResponse, error) {
	resp := &pb.RPCResponse{Data: "Hello " + in.GetAppName(), RPCName: name, Version: ver}
	rs := models.RequestResult{}
	//get input data into user session
	var usex models.UserSession
	usex.Session = in.Session
	usex.Action = in.Action
	usex.UserID = in.UserID
	usex.UserIP = in.UserIP
	usex.Params = in.Params

	if usex.Action == "l" {
		rs = login(usex)
	} else if usex.Action == "lo" {
		rs = logout(usex.Session)
	} else if usex.Action == "aut" {
		rs = auth(usex)
	} else {
		//unknow action
		return resp, nil
	}
	//convert RequestResult into json
	b, _ := json.Marshal(rs)
	resp.Data = string(b)
	return resp, nil
}

//auth: to authenticate user already login, for portal, return userid[+]shopid
func auth(usex models.UserSession) models.RequestResult {
	rs := rpch.GetLogin(usex.Session)
	if rs.UserId.Hex() == "" {
		return models.RequestResult{Error: "user not logged in"}
	} else {
		return models.RequestResult{Error: "", Status: 1, Data: `{"userid":"` + rs.UserId.Hex() + `","sex":"` + usex.Session + `","shop":"` + rs.ShopId + `"}`}
	}
}

//login user and update Session and IP in user_login. then return auth call to get userid[+]shopid
func login(usex models.UserSession) models.RequestResult {

	args := strings.Split(usex.Params, ",")
	if len(args) < 2 {
		return models.RequestResult{Error: "empty username or pass"}
	}
	user := args[0]
	pass := args[1]
	if rpch.Login(user, pass, usex.Session, usex.UserIP) {
		reply := auth(usex)
		if reply.Status == 1 {
			var rs map[string]string
			json.Unmarshal([]byte(reply.Data), &rs)
			return models.RequestResult{
				Error:   "",
				Status:  1,
				Message: "logged in",
				Data:    `{"sex":"` + rs["sex"] + `","shop":"` + rs["shop"] + `"}`}
		}
	}
	return models.RequestResult{Error: "Login failed"}

}

func logout(session string) models.RequestResult {
	rpch.Logout(session)
	return models.RequestResult{Error: "", Status: 1, Message: "Logout success"}

}
func main() {
	//default port for service
	var port string
	port = os.Getenv("PORT")
	if port == "" {
		port = "8901"
	}
	//open service and listen
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Errorf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	fmt.Printf("listening on %s\n", port)
	pb.RegisterGRPCServicesServer(s, &service{})
	if err := s.Serve(lis); err != nil {
		log.Errorf("failed to serve : %v", err)
	}
	fmt.Print("exit")

}
