package handler

import (
	"net/http"

	"github.com/frhan23/chat-go/res"
	"github.com/frhan23/chat-go/types"
)

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	data := types.Res{
		Message: "im alive",
	}
	res.ResOkJSON(w, &data)
}
