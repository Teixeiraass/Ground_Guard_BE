package db

import (
	"context"
	"testing"
	"time"

	"github.com/Teixeiraass/ground_guard_be/util"
	"github.com/stretchr/testify/require"
)

func createRandomUser(t *testing.T) User {
	hashedPassword, err := util.HashPassword(util.RandomString(6))
	require.NoError(t, err)

	arg := CreateUserParams{
		Username:       util.RandomUser(),
		HashedPassword: hashedPassword,
		FullName:       util.RandomUser(),
		Email:          util.RandomEmail(),
	}

	user, err := testQueries.CreateUser(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, user)

	require.Equal(t, arg.Username, user.Username)
	require.Equal(t, arg.HashedPassword, user.HashedPassword)
	require.Equal(t, arg.FullName, user.FullName)
	require.Equal(t, arg.Email, user.Email)

	require.True(t, user.PasswordChangedAt.IsZero())
	require.NotZero(t, user.CreatedAt)

	return user
}

func TestCreateUser(t *testing.T) {
	createRandomUser(t)
}

func TestGetUser(t *testing.T) {
	//create user
	user1 := createRandomUser(t)
	user2, err := testQueries.GetUser(context.Background(), user1.Uuid)
	require.NoError(t, err)
	require.NotEmpty(t, user2)

	require.Equal(t, user1.Username, user2.Username)
	require.Equal(t, user1.HashedPassword, user2.HashedPassword)
	require.Equal(t, user1.FullName, user2.FullName)
	require.Equal(t, user1.Email, user2.Email)
	require.Equal(t, user1.ProfileImage, user2.ProfileImage)
	require.Equal(t, user1.PasswordChangedAt, user2.PasswordChangedAt)
	require.WithinDuration(t, user1.CreatedAt, user2.CreatedAt, time.Second)
}

func TestGetUserByEmail(t *testing.T) {
	//create user
	user1 := createRandomUser(t)
	user2, err := testQueries.GetUserByEmail(context.Background(), user1.Email)
	require.NoError(t, err)
	require.NotEmpty(t, user2)

	require.Equal(t, user1.Username, user2.Username)
	require.Equal(t, user1.HashedPassword, user2.HashedPassword)
	require.Equal(t, user1.FullName, user2.FullName)
	require.Equal(t, user1.Email, user2.Email)
	require.Equal(t, user1.ProfileImage, user2.ProfileImage)
	require.Equal(t, user1.PasswordChangedAt, user2.PasswordChangedAt)
	require.WithinDuration(t, user1.CreatedAt, user2.CreatedAt, time.Second)
}

func TestUpdateUserName(t *testing.T) {
	user1 := createRandomUser(t)

	arg := UpdateUserNameParams{
		Uuid:     user1.Uuid,
		FullName: util.RandomString(10),
	}

	user2, err := testQueries.UpdateUserName(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, user2)

	require.Equal(t, user1.ID, user2.ID)
	require.Equal(t, user1.Uuid, user2.Uuid)
	require.Equal(t, user1.Username, user2.Username)
	require.Equal(t, user1.HashedPassword, user2.HashedPassword)
	require.Equal(t, arg.FullName, user2.FullName)
	require.Equal(t, user1.Email, user2.Email)
	require.Equal(t, user1.ProfileImage, user2.ProfileImage)
	require.WithinDuration(t, user1.PasswordChangedAt, user2.PasswordChangedAt, time.Second)
	require.WithinDuration(t, user1.CreatedAt, user2.CreatedAt, time.Second)
}

// ESSE TEST FUNCIONA, MAS FICA CRIANDO ARQUIVO EM UMA PASTA UPLOAD NO SQLC
// func TestUpdateProfileImage(t *testing.T) {
// 	user1 := createRandomUser(t)
// 	image, err := util.RandomImage()
// 	require.NoError(t, err)
// 	require.NotEmpty(t, image)

// 	pathImagem, err := util.SaveUserImage(image, user1.Username)
// 	require.NoError(t, err)
// 	require.NotEmpty(t, pathImagem)

// 	arg := UpdateUserProfileImageParams{
// 		Uuid: user1.Uuid,
// 		ProfileImage: sql.NullString{
// 			String: pathImagem,
// 			Valid: true,
// 		},
// 	}

// 	user2, err := testQueries.UpdateUserProfileImage(context.Background(), arg)
// 	require.NoError(t, err)
// 	require.NotEmpty(t, user2)

// 	require.Equal(t, user1.ID, user2.ID)
// 	require.Equal(t, user1.Uuid, user2.Uuid)
// 	require.Equal(t, user1.Username, user2.Username)
// 	require.Equal(t, user1.HashedPassword, user2.HashedPassword)
// 	require.Equal(t, user1.FullName, user2.FullName)
// 	require.Equal(t, user1.Email, user2.Email)
// 	require.Equal(t, arg.ProfileImage, user2.ProfileImage)
// 	require.WithinDuration(t, user1.PasswordChangedAt, user2.PasswordChangedAt, time.Second)
// 	require.WithinDuration(t, user1.CreatedAt, user2.CreatedAt, time.Second)
// }
