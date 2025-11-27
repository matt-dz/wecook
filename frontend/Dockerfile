FROM --platform=$BUILDPLATFORM node:24 AS build

WORKDIR /app
COPY package.json package-lock.json ./
COPY . .
ENV ADAPTER=node

RUN npm ci
RUN npm run build
RUN npm prune --production

FROM node:current-alpine
WORKDIR /app
COPY --from=build /app/build ./build/
COPY --from=build /app/package.json ./package.json
COPY --from=build /app/node_modules ./node_modules
EXPOSE 3000

CMD ["node", "build"]
