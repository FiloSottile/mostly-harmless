package nvram

import "log"

func Example() {
	res, err := Get("filippo")
	if err != nil {
		log.Fatalln(err.Error())
	}

	log.Printf("% x\n", res)

	err = Set("filippo", "\xff\x0042è\x00\xff")
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
