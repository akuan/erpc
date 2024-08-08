## canetproto 
canetproto is implemented CANET style socket communication protocol.
### Message Bytes
canet frame data format(Big Endian):

```sh
{1 byte frame info}
{4 bytes frame Id}
{8 bytes frame data(body)}
```

#### Frame Info
<table>
 <tr>
    <th>Bit</th>
    <th>7</th>
    <th>6</th>
    <th>5</th>
    <th>4</th>
    <th>3</th>
    <th>2</th>
    <th>1</th>
    <th>0</th>
  </tr>
  <tr>
    <td>Name</td>
    <td>FF=0</td>
    <td>RTP=0</td>
    <td>RS=0</td>
    <td>RS=0</td>
    <td colspan="4">Data len</td>
  </tr>
  <tr>
    <td>eg.</td>
    <td>0</td>
    <td>0</td>
    <td>0</td>
    <td>0</td>
    <td>1</td>
    <td>0</td>
    <td>0</td>
    <td>0</td>
  </tr>
</table>
 

### Usage



#### Test
 
