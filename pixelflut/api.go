package pixelflut

import (
	"bufio"
	"encoding/hex"
	"image"
	"image/color"
	"log"
	"net"
	"sync"
)

// Flut asynchronously sends the given image to pixelflut server at `address`
//   using `conns` connections. Pixels are sent row wise, unless `shuffle` is set.
// @cleanup: use FlutTask{} as arg
// @incomplete :cleanExit
func Flut(img image.Image, position image.Point, shuffle bool, address string, conns int, stop chan bool, wg *sync.WaitGroup) {
	cmds := commandsFromImage(img, position)
	if shuffle {
		cmds.Shuffle()
	}
	messages := cmds.Chunk(conns)

	bombWg := sync.WaitGroup{}
	for _, msg := range messages {
		bombWg.Add(1)
		go bombAddress(msg, address, stop, &bombWg)
	}
	bombWg.Wait()
	wg.Done()
}

// FetchImage asynchronously uses `conns` to fetch pixels within `bounds` from
//   a pixelflut server at `address`, and writes them into the returned Image.
func FetchImage(bounds image.Rectangle, address string, conns int, stop chan bool) (img *image.NRGBA) {
	img = image.NewNRGBA(bounds)
	// cmds := cmdsFetchImage(bounds).Chunk(conns)

	for i := 0; i < conns; i++ {
		conn, err := net.Dial("tcp", address)
		if err != nil {
			log.Fatal(err)
		}

		// @cleanup: parsePixels calls conn.Close(), as deferring it here would
		//   instantly close it
		go readPixels(img, conn, stop)
		// go bombConn(cmds[i], conn, stop)
	}

	return img
}

func readPixels(target *image.NRGBA, conn net.Conn, stop chan bool) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	col := make([]byte, 3)
	for {
		select {
		case <-stop:
			return

		default:
			res, err := reader.ReadSlice('\n')
			if err != nil {
				log.Fatal(err)
			}

			// parse response ("PX <x> <y> <col>\n")
			colorStart := len(res) - 7
			xy := res[3 : colorStart-1]
			yStart := 0
			for yStart = len(xy) - 2; yStart >= 0; yStart-- {
				if xy[yStart] == ' ' {
					break
				}
			}
			x := asciiToInt(xy[:yStart])
			y := asciiToInt(xy[yStart+1:])
			hex.Decode(col, res[colorStart:len(res)-1])

			target.SetNRGBA(x, y, color.NRGBA{col[0], col[1], col[2], 255})
		}
	}
}

func asciiToInt(buf []byte) (v int) {
	for _, c := range buf {
		v = v*10 + int(c-'0')
	}
	return v
}