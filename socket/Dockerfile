FROM node:22.4-slim
WORKDIR /app

ENV TZ=Asia/Ho_Chi_Minh
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime
RUN echo $TZ > /etc/timezone
RUN apt update

COPY main.mjs .
CMD ["node", "main.mjs"]
