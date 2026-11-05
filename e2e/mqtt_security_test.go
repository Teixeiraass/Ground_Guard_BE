//go:build e2e

package e2e

import (
	"encoding/json"
	"testing"
	"time"

	mqttgo "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/require"
)

func connectRaw(t *testing.T, clientID, user, pass string) (mqttgo.Client, error) {
	t.Helper()
	opts := mqttgo.NewClientOptions().AddBroker(cfg.MQTTBrokerURL).SetClientID(clientID).
		SetConnectTimeout(5 * time.Second).SetAutoReconnect(false)
	if user != "" {
		opts.SetUsername(user).SetPassword(pass)
	}
	c := mqttgo.NewClient(opts)
	tok := c.Connect()
	if !tok.WaitTimeout(5 * time.Second) {
		return nil, tok.Error()
	}
	if tok.Error() != nil {
		return nil, tok.Error()
	}
	t.Cleanup(func() { c.Disconnect(50) })
	return c, nil
}

// Attacks against the broker itself, from a client that is not the API.
func TestE2E_BrokerRefusesForgedCommandsAndSnooping(t *testing.T) {
	owner := newUser(t)
	victim := claimController(t, owner)

	attacker, err := connectRaw(t, randomDeviceUID(), "", "")
	require.NoError(t, err, "devices may still connect without credentials")

	t.Run("can't snoop telemetry or other devices' commands", func(t *testing.T) {
		// Mosquitto acknowledges the subscription but enforces read ACLs on
		// delivery, which is what matters: nothing must arrive.
		received := make(chan string, 16)
		for _, topic := range []string{topicPrefix + "/#", topicPrefix + "/devices/" + victim.UID + "/commands"} {
			tok := attacker.Subscribe(topic, 1, func(_ mqttgo.Client, m mqttgo.Message) { received <- m.Topic() })
			require.True(t, tok.WaitTimeout(5*time.Second))
		}

		victim.firmware.publish("telemetry", obj{"status": "ONLINE"})
		owner.api.expect("PUT", "/controller-devices/pairing/"+victim.UID, nil, 200, nil)
		victim.firmware.awaitCommand("PAIRING_START") // the real device still gets it

		select {
		case topic := <-received:
			t.Fatalf("attacker received a message on %s", topic)
		case <-time.After(1500 * time.Millisecond):
		}
	})

	t.Run("can't send commands to a controller", func(t *testing.T) {
		forged, _ := json.Marshal(obj{"command_id": "00000000-0000-0000-0000-000000000001", "action": "START"})
		tok := attacker.Publish(topicPrefix+"/devices/"+victim.UID+"/commands", 1, false, forged)
		tok.WaitTimeout(5 * time.Second)

		select {
		case cmd := <-victim.firmware.commands:
			t.Fatalf("forged command reached the controller: %v", cmd)
		case <-time.After(1500 * time.Millisecond):
		}
	})

	t.Run("can't post telemetry as another device", func(t *testing.T) {
		tok := attacker.Publish(topicPrefix+"/devices/"+victim.UID+"/state", 1, false, []byte(`{"status":"OFFLINE","is_online":false}`))
		tok.WaitTimeout(5 * time.Second)
		time.Sleep(1 * time.Second)
		var list []obj
		owner.api.expect("GET", "/controller-devices?page_id=1&page_size=10", nil, 200, &list)
		require.Equal(t, true, list[0]["is_online"], "fake OFFLINE from another client id must be dropped")
	})

	t.Run("wrong API password is rejected", func(t *testing.T) {
		_, err := connectRaw(t, "impostor-"+randomHex(3), cfg.MQTTUsername, "wrong-password")
		require.Error(t, err)
	})
}
