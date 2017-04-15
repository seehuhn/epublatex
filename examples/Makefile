SUBDIRS = $(shell find . -mindepth 2 -maxdepth 2 -name Makefile | sed 's;/[^/]*$$;;')
ALL = $(addsuffix .pdf,$(basename $(wildcard *.tex)))

.PHONY: all
all: $(ALL)
	@echo "output files:"
	@echo $(ALL)

%.pdf: %.tex handout.cls
	pdflatex $<

.PHONY: doclean clean dodistclean distclean
doclean:
	rm -f *.aux *.log *.dvi *.out *.synctex.gz
	rm -rf auto
	rm -f .RData .Rhistory
clean: doclean
	for d in $(SUBDIRS); do \
	    $(MAKE) -k -C $$d clean; \
	done
dodistclean: doclean
	rm -f $(ALL) *~
distclean: dodistclean
	for d in $(SUBDIRS); do \
	    $(MAKE) -k -C $$d distclean; \
	done
