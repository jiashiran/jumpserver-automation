FROM alpine:latest

ADD app /usr/local/
ADD static/ /Users/jiashiran/go/src/jumpserver-automation/static/
ADD key/ /Users/jiashiran/go/src/jumpserver-automation/build/key/

RUN chmod +x /usr/local/app

CMD /usr/local/app