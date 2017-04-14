package render

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"text/template"
)

const (
	imgNames = "img%d.png"

	queueLength = 1
	maxWorkers  = 4
)

var debugDir = flag.String("latex-render-debug", "",
	"directory to store rendering debugging information in")

// Queue allows to run LaTeX and to convert the output into images.
type Queue struct {
	resolution string
	workDir    string
	keepFiles  bool

	jobs    chan *jobSpec
	workers *sync.WaitGroup
}

// NewQueue creates a new rendering queue for converting .tex files to
// images.  The argument 'resolution' specifies the image resolution
// in pixels per inch.
func NewQueue(resolution int) (*Queue, error) {
	q := &Queue{
		resolution: strconv.Itoa(resolution),
		jobs:       make(chan *jobSpec, queueLength),
		workers:    &sync.WaitGroup{},
	}

	var workDir string
	var err error
	if *debugDir == "" {
		workDir, err = ioutil.TempDir("", "epublatex")
		if err != nil {
			return nil, err
		}
	} else {
		q.keepFiles = true
		workDir, err = filepath.Abs(*debugDir)
		if err != nil {
			return nil, err
		}
		log.Println("leaving rendering information in", workDir)
		err = os.MkdirAll(workDir, 0777)
		if err != nil {
			return nil, err
		}
	}
	q.workDir = workDir

	go q.scheduler()

	return q, nil
}

// Finish must be called after the last rendering job has been
// submitted to the queue.  The function waits until all rendered
// images have been delivered and then shuts down the queue.
func (q *Queue) Finish() error {
	close(q.jobs)
	q.workers.Wait()

	q.jobs = nil
	if !q.keepFiles {
		err := os.RemoveAll(q.workDir)
		if err != nil {
			return err
		}
	}
	return nil
}

func (q *Queue) scheduler() {
	workers := make(chan int, maxWorkers)
	for i := 0; i < maxWorkers; i++ {
		workers <- i
	}

	jobIdx := 1
	for job := range q.jobs {
		jobDir := filepath.Join(q.workDir, strconv.Itoa(jobIdx))
		jobIdx++

		worker := <-workers
		q.workers.Add(1)
		go func(job *jobSpec) {
			err := q.process(job, jobDir)
			workers <- worker
			if err != nil {
				log.Println(err)
			}
			q.workers.Done()
		}(job)
	}
}

// Submit adds a new rendering job to the queue.  As part of the job,
// the template tmpl is executed with the given data to obtain a TeX
// file.  Pdflatex is run with this file as input, and Ghostscript is
// used to convert each page of output into an image.  The resulting
// images can be read from the channel returned by .Submit().
func (q *Queue) Submit(tmpl *template.Template, data interface{}) <-chan image.Image {
	c := make(chan image.Image)
	job := &jobSpec{
		Template: tmpl,
		Data:     data,
		Result:   c,
	}
	q.jobs <- job
	return c
}

type jobSpec struct {
	Template *template.Template
	Data     interface{}
	Result   chan<- image.Image
}

func (q *Queue) process(job *jobSpec, jobDir string) (err error) {
	defer close(job.Result)

	err = os.MkdirAll(jobDir, 0777)
	if err != nil {
		return err
	}
	if !q.keepFiles {
		defer func() {
			e2 := os.RemoveAll(jobDir)
			if err == nil {
				err = e2
			}
		}()
	}

	// write TeX file
	texFileName := filepath.Join(jobDir, "job.tex")
	texFile, err := os.Create(texFileName)
	if err != nil {
		return err
	}
	err = job.Template.Execute(texFile, job.Data)
	if err != nil {
		return err
	}
	err = texFile.Close()
	if err != nil {
		return err
	}

	// convert to TeX -> PDF
	ltx := exec.Command("pdflatex", "-interaction=nonstopmode", "job.tex")
	ltx.Dir = jobDir
	output, err := ltx.Output()
	if err != nil {
		if e2, ok := err.(*exec.ExitError); ok {
			log.Println("Converting LaTeX to PDF failed:", e2)
			log.Println("--- begin LaTeX output ---")
			log.Println(string(output))
			log.Println("--- end LaTeX output ---")
		}
		return err
	}

	// convert to PDF -> PNG
	gs := exec.Command("gs", "-dSAFER", "-dBATCH", "-dNOPAUSE",
		"-r"+q.resolution, "-sDEVICE=pngalpha", "-dTextAlphaBits=4",
		"-sOutputFile="+imgNames, "job.pdf")
	gs.Dir = jobDir
	output, err = gs.Output()
	if err != nil {
		if e2, ok := err.(*exec.ExitError); ok {
			log.Println("Converting formulas to PNG failed:", e2)
			log.Println("--- begin gs output ---")
			log.Println(string(output))
			log.Println("--- end gs output ---")
		}
		return err
	}

	// read PNG, write to channel
	pageNo := 0
	for {
		pageNo++
		imageFileName := filepath.Join(jobDir, fmt.Sprintf(imgNames, pageNo))
		img, err := readImage(imageFileName)
		if os.IsNotExist(err) {
			break
		}
		if err != nil {
			log.Println("decoding image", imageFileName, "failed:", err)
		}
		job.Result <- img
	}

	return nil
}

func readImage(fname string) (image.Image, error) {
	fd, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer fd.Close()
	img, err := png.Decode(fd)
	if err != nil {
		return nil, err
	}
	return img, nil
}
