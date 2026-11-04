import type { CustomLayerInterface, Map as MapLibreMap } from "maplibre-gl"

const vertexSource = `
attribute vec2 a_position;
varying vec2 v_uv;
void main() {
  v_uv = a_position * 0.5 + 0.5;
  gl_Position = vec4(a_position, 0.0, 1.0);
}`

const fragmentSource = `
precision mediump float;
uniform sampler2D u_texture;
uniform float u_opacity;
varying vec2 v_uv;
void main() {
  gl_FragColor = texture2D(u_texture, vec2(v_uv.x, 1.0 - v_uv.y)) * u_opacity;
}`

function compile(gl: WebGL2RenderingContext, type: number, source: string) {
  const shader = gl.createShader(type)!
  gl.shaderSource(shader, source)
  gl.compileShader(shader)
  if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS)) {
    throw new Error(`Wind layer shader: ${gl.getShaderInfoLog(shader)}`)
  }
  return shader
}

/**
 * Draws the wind renderer's canvas inside the map, as a screen-sized
 * texture, so the particles sit in the map's layer order (over boundaries
 * and flood areas, under labels and gauges) rather than over everything.
 * The texture is uploaded only when the renderer has drawn a new frame.
 */
export class WindParticlesLayer implements CustomLayerInterface {
  readonly id: string
  readonly type = "custom" as const
  readonly renderingMode = "2d" as const
  private readonly source: HTMLCanvasElement
  private opacity: number
  private map: MapLibreMap | null = null
  private program: WebGLProgram | null = null
  private buffer: WebGLBuffer | null = null
  private texture: WebGLTexture | null = null
  private vertexArray: WebGLVertexArrayObject | null = null
  private positionLocation = -1
  private opacityLocation: WebGLUniformLocation | null = null
  private textureLocation: WebGLUniformLocation | null = null
  private dirty = true

  constructor(id: string, source: HTMLCanvasElement, opacity = 0.9) {
    this.id = id
    this.source = source
    this.opacity = opacity
  }

  /** A new frame is on the canvas: upload it and draw the map again. */
  invalidate() {
    this.dirty = true
    this.map?.triggerRepaint()
  }

  setOpacity(opacity: number) {
    this.opacity = opacity
    this.map?.triggerRepaint()
  }

  onAdd(map: MapLibreMap, gl: WebGL2RenderingContext) {
    this.map = map
    // MapLibre tracks GL state and does not expect it to change outside a render: put it back.
    const previous = {
      vertexArray: gl.getParameter(gl.VERTEX_ARRAY_BINDING) as WebGLVertexArrayObject | null,
      buffer: gl.getParameter(gl.ARRAY_BUFFER_BINDING) as WebGLBuffer | null,
      texture: gl.getParameter(gl.TEXTURE_BINDING_2D) as WebGLTexture | null,
    }
    const program = gl.createProgram()!
    gl.attachShader(program, compile(gl, gl.VERTEX_SHADER, vertexSource))
    gl.attachShader(program, compile(gl, gl.FRAGMENT_SHADER, fragmentSource))
    gl.linkProgram(program)
    if (!gl.getProgramParameter(program, gl.LINK_STATUS)) {
      throw new Error(`Wind layer program: ${gl.getProgramInfoLog(program)}`)
    }
    this.program = program
    this.positionLocation = gl.getAttribLocation(program, "a_position")
    this.opacityLocation = gl.getUniformLocation(program, "u_opacity")
    this.textureLocation = gl.getUniformLocation(program, "u_texture")

    // The layer's own vertex array, so MapLibre's are never touched.
    this.vertexArray = gl.createVertexArray()
    gl.bindVertexArray(this.vertexArray)
    this.buffer = gl.createBuffer()
    gl.bindBuffer(gl.ARRAY_BUFFER, this.buffer)
    gl.bufferData(
      gl.ARRAY_BUFFER,
      new Float32Array([-1, -1, 1, -1, -1, 1, -1, 1, 1, -1, 1, 1]),
      gl.STATIC_DRAW,
    )
    gl.enableVertexAttribArray(this.positionLocation)
    gl.vertexAttribPointer(this.positionLocation, 2, gl.FLOAT, false, 0, 0)

    this.texture = gl.createTexture()
    gl.bindTexture(gl.TEXTURE_2D, this.texture)
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
    this.dirty = true

    gl.bindVertexArray(previous.vertexArray)
    gl.bindBuffer(gl.ARRAY_BUFFER, previous.buffer)
    gl.bindTexture(gl.TEXTURE_2D, previous.texture)
  }

  onRemove(_map: MapLibreMap, gl: WebGL2RenderingContext) {
    if (this.program) gl.deleteProgram(this.program)
    if (this.buffer) gl.deleteBuffer(this.buffer)
    if (this.texture) gl.deleteTexture(this.texture)
    if (this.vertexArray) gl.deleteVertexArray(this.vertexArray)
    this.vertexArray = null
    this.program = null
    this.buffer = null
    this.texture = null
    this.map = null
  }

  render(gl: WebGL2RenderingContext) {
    if (!this.program || !this.texture || !this.vertexArray) return
    gl.activeTexture(gl.TEXTURE0)
    gl.bindTexture(gl.TEXTURE_2D, this.texture)
    if (this.dirty) {
      const premultiply = gl.getParameter(gl.UNPACK_PREMULTIPLY_ALPHA_WEBGL) as boolean
      const flip = gl.getParameter(gl.UNPACK_FLIP_Y_WEBGL) as boolean
      gl.pixelStorei(gl.UNPACK_PREMULTIPLY_ALPHA_WEBGL, true)
      gl.pixelStorei(gl.UNPACK_FLIP_Y_WEBGL, false)
      gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.RGBA, gl.UNSIGNED_BYTE, this.source)
      gl.pixelStorei(gl.UNPACK_PREMULTIPLY_ALPHA_WEBGL, premultiply)
      gl.pixelStorei(gl.UNPACK_FLIP_Y_WEBGL, flip)
      this.dirty = false
    }
    gl.useProgram(this.program)
    gl.bindVertexArray(this.vertexArray)
    gl.uniform1i(this.textureLocation, 0)
    gl.uniform1f(this.opacityLocation, this.opacity)
    gl.disable(gl.DEPTH_TEST)
    gl.disable(gl.STENCIL_TEST)
    gl.enable(gl.BLEND)
    gl.blendFunc(gl.ONE, gl.ONE_MINUS_SRC_ALPHA)
    gl.drawArrays(gl.TRIANGLES, 0, 6)
    gl.bindVertexArray(null)
  }
}
