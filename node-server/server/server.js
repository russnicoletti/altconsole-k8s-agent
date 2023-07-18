const express = require('express')
const app = express()
const bodyParser = require('body-parser')
const port = 3000

app.use(bodyParser.json({type: 'application/json', inflate: true}))
app.use(bodyParser.urlencoded({ extended: false }))

app.get('/', (req, res) => {
  console.log("processing root path")
  res.send('root path\n')
})

app.get('/foo', (req, res) => {
  console.log("processing 'foo' path")
  res.send('foo path\n')
})

app.post('/kubernetes/resource', (req, res) => {
  console.log()
  console.log("processing 'kubernetes/resource' path - request body: ")
  console.log(JSON.stringify(req.body))
  res.send('post recieved')
})

app.listen(port, () => {
  console.log(`nodeserver listening on port ${port}`)
})
