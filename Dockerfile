FROM quay.io/cybozu/ubuntu-debug:20.04

# https://docs.github.com/en/packages/managing-container-images-with-github-container-registry/connecting-a-repository-to-a-container-image#connecting-a-repository-to-a-container-image-on-the-command-line
LABEL org.opencontainers.image.source https://github.com/ymmt2005/pdf-converter

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        libreoffice \
        fonts-noto fonts-noto-cjk fonts-noto-cjk-extra \
        fonts-noto-color-emoji fonts-noto-extra fonts-noto-mono \
        fonts-noto-ui-core fonts-noto-ui-extra fonts-noto-unhinted \
        openjdk-11-jre-headless \
    && rm -rf /var/lib/apt/lists/*

RUN groupadd -g 10000 libre \
    && useradd -u 10000 -d /home/libre -g libre libre \
    && mkdir -p /home/libre/work \
    && chown -R 10000:10000 /home/libre

VOLUME /home/libre
WORKDIR /home/libre

USER 10000:10000
EXPOSE 8080

COPY pdf-converter /usr/local/bin/pdf-converter
CMD [ "/usr/local/bin/pdf-converter", "/home/libre/work" ]
