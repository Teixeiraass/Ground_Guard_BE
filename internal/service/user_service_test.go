package service_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	mockdb "github.com/Teixeiraass/ground_guard_be/db/mock"
	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/Teixeiraass/ground_guard_be/internal/service"
	"github.com/Teixeiraass/ground_guard_be/token"
	"github.com/Teixeiraass/ground_guard_be/util"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func newTestUserService(store db.Store) service.UserService {
	tokenMaker, err := token.NewPasetoMaker(util.RandomString(32))
	if err != nil {
		panic(err)
	}

	config := util.Config{AccessTokenDuration: time.Minute, RefreshTokenDuration: time.Hour}
	return service.NewUserService(store, tokenMaker, config)
}

func TestLoginUserRejectsBannedUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	password := util.RandomString(8)
	hashedPassword, err := util.HashPassword(password)
	require.NoError(t, err)

	user := db.User{
		ID:             1,
		Email:          util.RandomEmail(),
		HashedPassword: hashedPassword,
		Status:         "BANNED",
	}

	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetUserByEmail(gomock.Any(), user.Email).Return(user, nil)

	tokenMaker, err := token.NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)

	config := util.Config{AccessTokenDuration: time.Minute, RefreshTokenDuration: time.Hour}
	userService := service.NewUserService(store, tokenMaker, config)

	_, err = userService.LoginUser(context.Background(), dto.LoginUserRequest{
		Email:    user.Email,
		Password: password,
	}, "test-agent", "127.0.0.1")

	require.ErrorIs(t, err, service.ErrUserBanned)
}

func TestLoginUserAllowsActiveUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	password := util.RandomString(8)
	hashedPassword, err := util.HashPassword(password)
	require.NoError(t, err)

	user := db.User{
		ID:             1,
		Email:          util.RandomEmail(),
		HashedPassword: hashedPassword,
		Status:         "ACTIVE",
	}

	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetUserByEmail(gomock.Any(), user.Email).Return(user, nil)
	store.EXPECT().CreateSession(gomock.Any(), gomock.Any()).Return(db.Session{}, nil)

	tokenMaker, err := token.NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)

	config := util.Config{AccessTokenDuration: time.Minute, RefreshTokenDuration: time.Hour}
	userService := service.NewUserService(store, tokenMaker, config)

	rsp, err := userService.LoginUser(context.Background(), dto.LoginUserRequest{
		Email:    user.Email,
		Password: password,
	}, "test-agent", "127.0.0.1")

	require.NoError(t, err)
	require.NotNil(t, rsp)
}

func TestLoginUserRejectsDeletedUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	password := util.RandomString(8)
	hashedPassword, err := util.HashPassword(password)
	require.NoError(t, err)

	user := db.User{
		ID:             1,
		Email:          util.RandomEmail(),
		HashedPassword: hashedPassword,
		Status:         "DELETED",
	}

	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetUserByEmail(gomock.Any(), user.Email).Return(user, nil)

	userService := newTestUserService(store)

	_, err = userService.LoginUser(context.Background(), dto.LoginUserRequest{
		Email:    user.Email,
		Password: password,
	}, "test-agent", "127.0.0.1")

	require.Error(t, err)
	require.NotErrorIs(t, err, service.ErrUserBanned)
}

func TestLoginUserRecordsDeviceInfo(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	password := util.RandomString(8)
	hashedPassword, err := util.HashPassword(password)
	require.NoError(t, err)

	user := db.User{
		ID:             1,
		Uuid:           uuid.New(),
		Email:          util.RandomEmail(),
		HashedPassword: hashedPassword,
		Status:         "ACTIVE",
	}

	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetUserByEmail(gomock.Any(), user.Email).Return(user, nil)
	store.EXPECT().CreateSession(gomock.Any(), gomock.Any()).Return(db.Session{}, nil)
	store.EXPECT().
		UpdateUserDeviceInfo(gomock.Any(), gomock.Any()).
		Do(func(_ context.Context, arg db.UpdateUserDeviceInfoParams) {
			require.Equal(t, user.Uuid, arg.Uuid)
			require.Equal(t, "iPhone 15", arg.LastDeviceModel.String)
			require.Equal(t, "ios", arg.LastPlatform.String)
		}).
		Return(user, nil)

	userService := newTestUserService(store)

	_, err = userService.LoginUser(context.Background(), dto.LoginUserRequest{
		Email:       user.Email,
		Password:    password,
		DeviceModel: "iPhone 15",
		Platform:    "ios",
		OsVersion:   "17.0",
		AppVersion:  "1.2.0",
	}, "test-agent", "127.0.0.1")

	require.NoError(t, err)
}

func TestChangePasswordRejectsWrongCurrentPassword(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	hashedPassword, err := util.HashPassword("correct-password")
	require.NoError(t, err)

	user := db.User{ID: 1, Uuid: uuid.New(), HashedPassword: hashedPassword}

	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetUser(gomock.Any(), user.Uuid).Return(user, nil)

	userService := newTestUserService(store)

	err = userService.ChangePassword(context.Background(), user.Uuid, "wrong-password", "new-password")
	require.ErrorIs(t, err, service.ErrCurrentPasswordIncorrect)
}

func TestChangePasswordSucceedsAndPersistsNewHash(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	hashedPassword, err := util.HashPassword("correct-password")
	require.NoError(t, err)

	user := db.User{ID: 1, Uuid: uuid.New(), HashedPassword: hashedPassword}

	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetUser(gomock.Any(), user.Uuid).Return(user, nil)
	store.EXPECT().
		UpdateUserPassword(gomock.Any(), gomock.Any()).
		Do(func(_ context.Context, arg db.UpdateUserPasswordParams) {
			require.Equal(t, user.Uuid, arg.Uuid)
			require.NoError(t, util.CheckPassword("new-password", arg.HashedPassword))
		}).
		Return(user, nil)

	userService := newTestUserService(store)

	err = userService.ChangePassword(context.Background(), user.Uuid, "correct-password", "new-password")
	require.NoError(t, err)
}

func TestDeleteAccountSetsDeletedAndRevokesSessions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	hashedPassword, err := util.HashPassword("correct-password")
	require.NoError(t, err)

	user := db.User{ID: 1, Uuid: uuid.New(), HashedPassword: hashedPassword}

	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetUser(gomock.Any(), user.Uuid).Return(user, nil)
	store.EXPECT().SoftDeleteUser(gomock.Any(), user.Uuid).Return(user, nil)
	store.EXPECT().BlockAllSessionsForUser(gomock.Any(), user.ID).Return(nil)

	userService := newTestUserService(store)

	err = userService.DeleteAccount(context.Background(), user.Uuid, "correct-password")
	require.NoError(t, err)
}

func TestDeleteAccountRejectsWrongPassword(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	hashedPassword, err := util.HashPassword("correct-password")
	require.NoError(t, err)

	user := db.User{ID: 1, Uuid: uuid.New(), HashedPassword: hashedPassword}

	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetUser(gomock.Any(), user.Uuid).Return(user, nil)

	userService := newTestUserService(store)

	err = userService.DeleteAccount(context.Background(), user.Uuid, "wrong-password")
	require.ErrorIs(t, err, service.ErrCurrentPasswordIncorrect)
}

func TestVerifyEmailValidToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().
		VerifyUserEmailByToken(gomock.Any(), gomock.Any()).
		Return(db.User{ID: 1}, nil)

	userService := newTestUserService(store)

	err := userService.VerifyEmail(context.Background(), "some-raw-token")
	require.NoError(t, err)
}

func TestVerifyEmailExpiredOrUnknownToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().
		VerifyUserEmailByToken(gomock.Any(), gomock.Any()).
		Return(db.User{}, sql.ErrNoRows)

	userService := newTestUserService(store)

	err := userService.VerifyEmail(context.Background(), "some-raw-token")
	require.ErrorIs(t, err, service.ErrInvalidOrExpiredToken)
}

func TestRequestPasswordResetIsSilentWhenEmailDoesNotExist(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetUserByEmail(gomock.Any(), "nobody@example.com").Return(db.User{}, sql.ErrNoRows)

	userService := newTestUserService(store)

	err := userService.RequestPasswordReset(context.Background(), "nobody@example.com", "", "127.0.0.1")
	require.NoError(t, err)
}

func TestRequestPasswordResetCreatesTokenWhenEmailExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	user := db.User{ID: 1, Uuid: uuid.New(), Email: util.RandomEmail()}

	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetUserByEmail(gomock.Any(), user.Email).Return(user, nil)
	store.EXPECT().
		CreatePasswordResetToken(gomock.Any(), gomock.Any()).
		Do(func(_ context.Context, arg db.CreatePasswordResetTokenParams) {
			require.Equal(t, user.ID, arg.UserID)
			require.NotEmpty(t, arg.TokenHash)
		}).
		Return(db.PasswordResetToken{}, nil)

	userService := newTestUserService(store)

	err := userService.RequestPasswordReset(context.Background(), user.Email, "", "127.0.0.1")
	require.NoError(t, err)
}

func TestConfirmPasswordResetRejectsInvalidToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().
		ConsumePasswordResetToken(gomock.Any(), gomock.Any()).
		Return(db.PasswordResetToken{}, sql.ErrNoRows)

	userService := newTestUserService(store)

	err := userService.ConfirmPasswordReset(context.Background(), "bad-token", "new-password", "", "127.0.0.1")
	require.ErrorIs(t, err, service.ErrInvalidOrExpiredToken)
}

func TestConfirmPasswordResetUpdatesPasswordAndRevokesSessions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	user := db.User{ID: 1, Uuid: uuid.New(), Email: util.RandomEmail()}
	resetToken := db.PasswordResetToken{ID: 10, UserID: user.ID}

	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().ConsumePasswordResetToken(gomock.Any(), gomock.Any()).Return(resetToken, nil)
	store.EXPECT().GetUserByID(gomock.Any(), user.ID).Return(user, nil)
	store.EXPECT().
		UpdateUserPassword(gomock.Any(), gomock.Any()).
		Do(func(_ context.Context, arg db.UpdateUserPasswordParams) {
			require.Equal(t, user.Uuid, arg.Uuid)
			require.NoError(t, util.CheckPassword("new-password", arg.HashedPassword))
		}).
		Return(user, nil)
	store.EXPECT().InvalidatePasswordResetTokensForUser(gomock.Any(), user.ID).Return(nil)
	store.EXPECT().BlockAllSessionsForUser(gomock.Any(), user.ID).Return(nil)

	userService := newTestUserService(store)

	err := userService.ConfirmPasswordReset(context.Background(), "raw-token", "new-password", "", "127.0.0.1")
	require.NoError(t, err)
}
