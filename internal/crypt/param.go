package crypt

import "crypto/hpke"

func KEM() hpke.KEM {
	return hpke.MLKEM768X25519()
}

func KDF() hpke.KDF {
	return hpke.HKDFSHA256()
}

func AEAD() hpke.AEAD {
	return hpke.AES256GCM()
}

var Info = []byte("korob/hpke/v1")
