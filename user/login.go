package user

import (
	"database/sql"
	"errors"
	"fmt"
)

type Credentials struct {
	UserId   string `json:"UserId"`
	Password string `json:"Password"`
}

type LoginResponse struct {
	AliceClientId      string `json:"AliceClientId"`
	AliceApiKey        string `json:"AliceApiKey"`
	AngelOneClientCode string `json:"AngelOneClientCode"`
	AngelOnePassword   string `json:"AngelOnePassword"`
	AngelOneMarketKey  string `json:"AngelOneMarketKey"`
	Name               string `json:"Name"`
}

type Service struct {
	DB *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{DB: db}
}

func (s *Service) LoginUser(creds Credentials) (*LoginResponse, error) {
	// Step 1: Verify credentials from Auth table using DB.Query
	authQuery := fmt.Sprintf(`
		SELECT "UserId" FROM "User"."Auth"
		WHERE "UserId" = '%s' AND "Password" = '%s'`,
		creds.UserId, creds.Password,
	)
	//authQuery := `SELECT "UserId" FROM "User"."Auth" WHERE "UserId" = 'pritivardhan.856@gmail.com' AND "Password" = 'Prit@2326'`

	fmt.Println("Running Auth Query:", authQuery)

	rows, err := s.DB.Query(authQuery)
	if err != nil {
		fmt.Println("Error:", err.Error())
		return nil, fmt.Errorf("query failed: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, errors.New("invalid user ID or password")
	}

	var userId string
	if err := rows.Scan(&userId); err != nil {
		return nil, fmt.Errorf("scan failed: %v", err)
	}

	// Step 2: Fetch user's trading credentials using DB.Query
	userQuery := fmt.Sprintf(`
		SELECT "AliceClientId", "AliceApiKey", "AngelOneClientCode", "AngelOnePassword", "AngelOneMarketKey", "Name"
		FROM "User"."User"
		WHERE "UserId" = '%s'`, creds.UserId)

	userRows, err := s.DB.Query(userQuery)
	if err != nil {
		return nil, fmt.Errorf("user query failed: %v", err)
	}
	defer userRows.Close()

	if !userRows.Next() {
		return nil, errors.New("user not found in user table")
	}

	var res LoginResponse
	if err := userRows.Scan(
		&res.AliceClientId,
		&res.AliceApiKey,
		&res.AngelOneClientCode,
		&res.AngelOnePassword,
		&res.AngelOneMarketKey,
		&res.Name,
	); err != nil {
		return nil, fmt.Errorf("user scan failed: %v", err)
	}

	return &res, nil
}
