package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"svc-opt/lib/batcher"
	"svc-opt/lib/ioc"
	"svc-opt/lib/model"
)

type CallUserServiceBatchResp struct {
	User *model.User
	Err  error
}

// GetUserByID handles the request to fetch a user by their ID.
func GetUserByID(w http.ResponseWriter, r *http.Request) {
	re := regexp.MustCompile(`/user/(\d+)`)
	matches := re.FindStringSubmatch(r.URL.Path)
	if len(matches) != 2 {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(matches[1])
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Call the user service to fetch the user
	user, err := CallUserService(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// fmt.Println("GetUserByID: ", "request", userID, "name", user.Name)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// GetUserByBatch handles the request to fetch a user by their ID.
func GetUserByIDBatch(w http.ResponseWriter, r *http.Request) {
	re := regexp.MustCompile(`/user-batch/(\d+)`)
	matches := re.FindStringSubmatch(r.URL.Path)
	if len(matches) != 2 {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(matches[1])
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	request := batcher.NewRequest(ioc.GET_USER_BY_IDS, "ID", userID)
	if err := ioc.BatchIOC.AddRequest(request); err != nil {
		http.Error(w, "Error batch add request", http.StatusInternalServerError)
		return
	}

	result := <-request.ResultChan
	resp := result.(*CallUserServiceBatchResp)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp.User)
}

// CallUserService makes an HTTP request to the user service.
func CallUserService(id int) (*model.User, error) {
	resp, err := http.Get("http://localhost:8081/user/" + strconv.Itoa(id))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user not found")
	}

	var users []*model.User
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, err
	}

	if len(users) == 0 {
		return nil, nil
	}

	return (users)[0], nil
}

// CallUserServiceBatch makes an HTTP request to the user service.
func CallUserServiceBatch(idList string) ([]*model.User, error) {
	resp, err := http.Get("http://localhost:8081/user/" + idList)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user not found")
	}

	var users []*model.User
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, err
	}

	if len(users) == 0 {
		return nil, nil
	}

	return users, nil
}

var BatchRunner = func(ctx context.Context, reqs []*batcher.Request, done chan struct{}) {
	idStrs := []string{}
	for _, req := range reqs {
		idStrs = append(idStrs, strconv.Itoa(req.AggregatedValue.(int)))
	}
	idList := strings.Join(idStrs, ",")

	users, err := CallUserServiceBatch(idList)
	resultMap := make(map[int]*model.User)
	for _, user := range users {
		resultMap[user.ID] = user
	}

	for _, req := range reqs {
		req.ResultChan <- &CallUserServiceBatchResp{
			User: resultMap[req.AggregatedValue.(int)],
			Err:  err,
		}
		close(req.ResultChan)
	}
	done <- struct{}{}
}
