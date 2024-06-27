package protocol

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"io"
	"net"
	"strconv"
)

const (
	StateKeyExchange VarInt = 0x01 // payload is encrypted with Wired Publickey (default)
	StateReady       VarInt = 0x02 // switched to AES cipherstream
)

type Conn struct {
	Address net.IP
	Port    uint16
	State   VarInt
	conn    net.Conn
	r       io.Reader
	w       io.Writer
}

type RSAStream struct {
	key  *rsa.PrivateKey
	pub  *rsa.PublicKey
	conn net.Conn
}

func (r *RSAStream) Read(p []byte) (n int, err error) { // should be used for key exchange state
	var l VarInt
	_, err = l.ReadFrom(r.conn)
	if err != nil {
		return 0, err
	}

	buf := make([]byte, l)
	bytesRead, err := io.ReadFull(r.conn, buf)
	if err != nil {
		return 0, err
	}
	decrypted, err := rsa.DecryptPKCS1v15(rand.Reader, r.key, buf[:bytesRead])
	if err != nil {
		return 0, err
	}

	copy(p, decrypted)

	return len(decrypted), err
}

func (r *RSAStream) Write(p []byte) (n int, err error) {
	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, r.pub, p)
	if err != nil {
		return 0, err
	}

	l := VarInt(len(encrypted))
	_, err = l.WriteTo(r.conn)
	if err != nil {
		return 0, err
	}

	_, err = r.conn.Write(encrypted)
	return len(p), err
}

func NewConn(c net.Conn, wiredKey *rsa.PrivateKey, wiredPub *rsa.PublicKey) *Conn {
	addr, portStr, _ := net.SplitHostPort(c.RemoteAddr().String())

	port, _ := strconv.Atoi(portStr)

	reader := io.Reader(c)
	if wiredKey != nil {
		reader = &RSAStream{key: wiredKey, conn: c}
	}

	writer := io.Writer(c)
	if wiredPub != nil {
		writer = &RSAStream{pub: wiredPub, conn: c}
	}

	return &Conn{
		Address: net.ParseIP(addr),
		Port:    uint16(port),
		conn:    c,
		State:   StateKeyExchange,
		r:       reader,
		w:       writer,
	}
}

func (c *Conn) EnableEncryption(sharedSecret []byte) error {
	block, err := aes.NewCipher(sharedSecret)
	if err != nil {
		return err
	}
	c.setCipher(NewCFB8Encrypter(block, sharedSecret), NewCFB8Decrypter(block, sharedSecret))
	c.State = StateReady
	return nil
}

func (c *Conn) setCipher(encStream, decStream cipher.Stream) {
	c.r = cipher.StreamReader{
		S: decStream,
		R: c.conn,
	}
	c.w = cipher.StreamWriter{
		S: encStream,
		W: c.conn,
	}
}

func (c *Conn) Read(p []byte) (n int, err error) {
	return c.r.Read(p)
}

func (c *Conn) Write(p []byte) (n int, err error) {
	return c.w.Write(p)
}

func (c *Conn) Close() error {
	return c.conn.Close()
}
func (c *Conn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *Conn) GetSocket() net.Conn {
	return c.conn
}

func (c *Conn) SendPacket(id VarInt, packet any) error {
	//marshal packet
	data, err := EncodePacket(packet)
	if err != nil {
		return err
	}

	//assemble and send packet
	_, err = (&Packet{
		ID:   id,
		Data: data,
	}).Write(c)
	return err
}

func MarshalPacket(id VarInt, packet any) ([]byte, error) {
	var data []byte
	var err error
	if packet != nil {
		data, err = EncodePacket(packet)
		if err != nil {
			return nil, err
		}
	}

	buf := bytes.Buffer{}
	_, err = (&Packet{
		ID:   id,
		Data: data,
	}).Write(&buf)
	return buf.Bytes(), err
}
