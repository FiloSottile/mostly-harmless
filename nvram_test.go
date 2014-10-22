package nvram

import "log"

func Example() {
	err := Setup()
	if err != nil {
		log.Fatalln(err.Error())
	}
	defer Teardown()

	res, err := Get("filippo")
	if err != nil {
		log.Fatalln(err.Error())
	}

	log.Printf("% x\n", res)

	err = Set("filippo", "\x0042è\x00\xff")
	if err != nil {
		log.Fatalln(err.Error())
	}

	log.Println("Set done")

	res, err = Get("filippo")
	if err != nil {
		log.Fatalln(err.Error())
	}

	log.Printf("% x\n", res)
}
