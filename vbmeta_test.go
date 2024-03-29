package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"testing"
)

const (
	/* Size of the vbmeta image header. */
	AVB_VBMETA_IMAGE_HEADER_SIZE = 256

	/* Magic for the vbmeta image header. */
	AVB_MAGIC     = "AVB0"
	AVB_MAGIC_LEN = 4

	/* Maximum size of the release string including the terminating NUL byte. */
	AVB_RELEASE_STRING_SIZE = 48

	/* Flags for the vbmeta image.
	 *
	 * AVB_VBMETA_IMAGE_FLAGS_HASHTREE_DISABLED: If this flag is set,
	 * hashtree image verification will be disabled.
	 *
	 * AVB_VBMETA_IMAGE_FLAGS_VERIFICATION_DISABLED: If this flag is set,
	 * verification will be disabled and descriptors will not be parsed.
	 */
	AVB_VBMETA_IMAGE_FLAGS_HASHTREE_DISABLED     = 1 << 0
	AVB_VBMETA_IMAGE_FLAGS_VERIFICATION_DISABLED = 1 << 1
)

/* Binary format for header of the vbmeta image.
 *
 * The vbmeta image consists of three blocks:
 *
 *  +-----------------------------------------+
 *  | Header data - fixed size                |
 *  +-----------------------------------------+
 *  | Authentication data - variable size     |
 *  +-----------------------------------------+
 *  | Auxiliary data - variable size          |
 *  +-----------------------------------------+
 *
 * The "Header data" block is described by this struct and is always
 * |AVB_VBMETA_IMAGE_HEADER_SIZE| bytes long.
 *
 * The "Authentication data" block is |authentication_data_block_size|
 * bytes long and contains the hash and signature used to authenticate
 * the vbmeta image. The type of the hash and signature is defined by
 * the |algorithm_type| field.
 *
 * The "Auxiliary data" is |auxiliary_data_block_size| bytes long and
 * contains the auxiliary data including the public key used to make
 * the signature and descriptors.
 *
 * The public key is at offset |public_key_offset| with size
 * |public_key_size| in this block. The size of the public key data is
 * defined by the |algorithm_type| field. The format of the public key
 * data is described in the |AvbRSAPublicKeyHeader| struct.
 *
 * The descriptors starts at |descriptors_offset| from the beginning
 * of the "Auxiliary Data" block and take up |descriptors_size|
 * bytes. Each descriptor is stored as a |AvbDescriptor| with tag and
 * number of bytes following. The number of descriptors can be
 * determined by walking this data until |descriptors_size| is
 * exhausted.
 *
 * The size of each of the "Authentication data" and "Auxiliary data"
 * blocks must be divisible by 64. This is to ensure proper alignment.
 *
 * Descriptors are free-form blocks stored in a part of the vbmeta
 * image subject to the same integrity checks as the rest of the
 * image. See the documentation for |AvbDescriptor| for well-known
 * descriptors. See avb_descriptor_foreach() for a convenience
 * function to iterate over descriptors.
 *
 * This struct is versioned, see the |required_libavb_version_major|
 * and |required_libavb_version_minor| fields. This represents the
 * minimum version of libavb required to verify the header and depends
 * on the features (e.g. algorithms, descriptors) used. Note that this
 * may be 1.0 even if generated by an avbtool from 1.4 but where no
 * features introduced after 1.0 has been used. See the "Versioning
 * and compatibility" section in the README.md file for more details.
 *
 * All fields are stored in network byte order when serialized. To
 * generate a copy with fields swapped to native byte order, use the
 * function avb_vbmeta_image_header_to_host_byte_order().
 *
 * Before reading and/or using any of this data, you MUST verify it
 * using avb_vbmeta_image_verify() and reject it unless it's signed by
 * a known good public key.
 */

type AvbVBMetaImageHeader struct {
	/*   0: Four bytes equal to "AVB0" (AVB_MAGIC). */
	Magic [AVB_MAGIC_LEN]uint8

	/*   4: The major version of libavb required for this header. */
	Required_libavb_version_major uint32
	/*   8: The minor version of libavb required for this header. */
	Required_libavb_version_minor uint32

	/*  12: The size of the signature block. */
	Authentication_data_block_size uint64
	/*  20: The size of the auxiliary data block. */
	Auxiliary_data_block_size uint64

	/*  28: The verification algorithm used, see |AvbAlgorithmType| enum. */
	Algorithm_type uint32

	/*  32: Offset into the "Authentication data" block of hash data. */
	Hash_offset uint64
	/*  40: Length of the hash data. */
	Hash_size uint64

	/*  48: Offset into the "Authentication data" block of signature data. */
	Signature_offset uint64
	/*  56: Length of the signature data. */
	Signature_size uint64

	/*  64: Offset into the "Auxiliary data" block of public key data. */
	Public_key_offset uint64
	/*  72: Length of the public key data. */
	Public_key_size uint64

	/*  80: Offset into the "Auxiliary data" block of public key metadata. */
	Public_key_metadata_offset uint64
	/*  88: Length of the public key metadata. Must be set to zero if there
	 *  is no public key metadata.
	 */
	Public_key_metadata_size uint64

	/*  96: Offset into the "Auxiliary data" block of descriptor data. */
	Descriptors_offset uint64
	/* 104: Length of descriptor data. */
	Descriptors_size uint64

	/* 112: The rollback index which can be used to prevent rollback to
	 *  older versions.
	 */
	Rollback_index uint64

	/* 120: Flags from the AvbVBMetaImageFlags enumeration. This must be
	 * set to zero if the vbmeta image is not a top-level image.
	 */
	Flags uint32

	/* 124: The location of the rollback index defined in this header.
	 * Only valid for the main vbmeta. For chained partitions, the rollback
	 * index location must be specified in the AvbChainPartitionDescriptor
	 * and this value must be set to 0.
	 */
	Rollback_index_location uint32

	/* 128: The release string from avbtool, e.g. "avbtool 1.0.0" or
	 * "avbtool 1.0.0 xyz_board Git-234abde89". Is guaranteed to be NUL
	 * terminated. Applications must not make assumptions about how this
	 * string is formatted.
	 */
	Release_string [AVB_RELEASE_STRING_SIZE]uint8

	/* 176: Padding to ensure struct is size AVB_VBMETA_IMAGE_HEADER_SIZE
	 * bytes. This must be set to zeroes.
	 */
	Reserved [80]uint8
}

func (header *AvbVBMetaImageHeader) Read(buf []byte) error {
	err := binary.Read(bytes.NewReader(buf[0:AVB_VBMETA_IMAGE_HEADER_SIZE]), binary.LittleEndian, header)
	if err != nil {
		fmt.Println("binary.Read failed:", err)
	}
	return nil
}

func (header *AvbVBMetaImageHeader) Dump() {
	fmt.Printf("%+v", *header)
}

func TestVerityEnabled(t *testing.T) {
	if header.Flags&AVB_VBMETA_IMAGE_FLAGS_HASHTREE_DISABLED != 0 {
		t.Error("hashtree image verification is disabled")
	}
	if header.Flags&AVB_VBMETA_IMAGE_FLAGS_VERIFICATION_DISABLED != 0 {
		t.Error("verification is disabled and descriptors will not be parsed")
	}
	if header.Flags != 0 {
		t.Errorf("flags:0x%x vbmeta.img的flags错误", header.Flags)
	}
}

func TestZteAvbKey(t *testing.T) {
	fmt.Println("sdf")
}

var header AvbVBMetaImageHeader

func TestMain(m *testing.M) {
	fd, err := os.Open("vbmeta.img")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer fd.Close()

	buf := make([]byte, AVB_VBMETA_IMAGE_HEADER_SIZE)
	count, err := fd.Read(buf)
	if err != nil {
		fmt.Println(err)
		return
	}

	if count < AVB_VBMETA_IMAGE_HEADER_SIZE {
		fmt.Printf("bmeta image 数据读取错误，读取的header小于%v字节\n", AVB_VBMETA_IMAGE_HEADER_SIZE)
		return
	}

	header.Read(buf)
	//header.Dump()

	retCode := m.Run()

	os.Exit(retCode)
}
